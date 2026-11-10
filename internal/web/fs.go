package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tommilevs/keenetic-smb-ui/internal/acl"
)

// fsEntry is one row of the «Файлы и папки» list: a directory (or a file) of
// the aggregated share root.
type fsEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Dir         bool   `json:"dir"`
	Remote      bool   `json:"remote"` // network mount: sizes are not computed
	Size        int64  `json:"size"`   // -1 = not known yet
	HasChildren bool   `json:"hasChildren"`
}

// sizeState is the cached recursive size of a single directory. Sizes are
// expensive (the aggregated tree spans tens of terabytes and may live on CIFS),
// so they are computed by a single background worker and served from cache.
type sizeState struct {
	Bytes     int64     `json:"bytes"`
	Partial   bool      `json:"partial"`
	Ready     bool      `json:"ready"`
	Computing bool      `json:"computing"`
	At        time.Time `json:"at"`
}

const (
	sizeTTL    = 10 * time.Minute
	sizeBudget = 45 * time.Second
)

var (
	sizeMu    sync.Mutex
	sizeItems = map[string]*sizeState{}
	sizeQueue = make(chan string, 256)
	sizeOnce  sync.Once
)

func startSizeWorker() {
	sizeOnce.Do(func() {
		go func() {
			for p := range sizeQueue {
				bytes, partial := treeSize(p)
				sizeMu.Lock()
				st := sizeItems[p]
				if st == nil {
					st = &sizeState{}
					sizeItems[p] = st
				}
				st.Bytes = bytes
				st.Partial = partial
				st.Ready = true
				st.Computing = false
				st.At = time.Now()
				sizeMu.Unlock()
			}
		}()
	})
}

// treeSize walks a directory tree and returns the sum of file sizes plus a flag
// telling whether the walk hit the time budget. Errors on individual entries are
// ignored: a partially readable tree still gives a useful number. -1 means the
// walk itself failed.
func treeSize(root string) (int64, bool) {
	started := time.Now()
	target, err := filepath.EvalSymlinks(root)
	if err != nil {
		target = root
	}
	var total int64
	partial := false
	walkErr := filepath.WalkDir(target, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if time.Since(started) > sizeBudget {
			partial = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		if info, e := d.Info(); e == nil {
			total += info.Size()
		}
		return nil
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return -1, false
	}
	return total, partial
}

// requestSize returns the current size state of a directory and schedules a
// computation when nothing fresh is cached.
func requestSize(abs string) sizeState {
	sizeMu.Lock()
	defer sizeMu.Unlock()
	st := sizeItems[abs]
	if st != nil && st.Ready && time.Since(st.At) < sizeTTL {
		return *st
	}
	if st == nil {
		st = &sizeState{}
		sizeItems[abs] = st
	}
	if !st.Computing {
		st.Computing = true
		select {
		case sizeQueue <- abs:
		default:
			st.Computing = false
		}
	}
	return *st
}

func cachedSize(abs string) int64 {
	sizeMu.Lock()
	defer sizeMu.Unlock()
	if st := sizeItems[abs]; st != nil && st.Ready && time.Since(st.At) < sizeTTL {
		return st.Bytes
	}
	return -1
}

// shareAbs maps a share-relative path to an absolute one, refusing paths that
// escape the aggregated root.
func (s *Server) shareAbs(rel string) (string, error) {
	root := filepath.Clean(s.shares.Root())
	rel = acl.CleanPath(rel)
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", os.ErrPermission
	}
	return abs, nil
}

// remoteRoots returns the mount points of network filesystems (CIFS/NFS) that
// live inside the aggregated share. Sizes are never computed for them: walking
// a 24 TB CIFS tree on the router would hammer the Proxmox hosts for hours.
func (s *Server) remoteRoots() []string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		// not Linux (local development): fall back to the configured remotes
		out := make([]string, 0, len(s.cfg.Remotes))
		for _, r := range s.cfg.Remotes {
			out = append(out, filepath.Clean(s.shares.MountPoint(r)))
		}
		return out
	}
	root := filepath.Clean(s.shares.Root())
	out := make([]string, 0, 8)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		typ := fields[2]
		if typ != "cifs" && !strings.HasPrefix(typ, "nfs") && typ != "smb3" {
			continue
		}
		target := filepath.Clean(unescapeMount(fields[1]))
		if target == root || strings.HasPrefix(target, root+string(os.PathSeparator)) {
			out = append(out, target)
		}
	}
	return out
}

func unescapeMount(p string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(p)
}

// isRemotePath reports whether abs sits inside a network mount of the share.
func (s *Server) isRemotePath(abs string) bool {
	abs = filepath.Clean(abs)
	for _, r := range s.remoteRoots() {
		if abs == r || strings.HasPrefix(abs, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// dirLike reports whether an entry is a directory, following symlinks (the
// aggregated root links local router disks as local-<uuid>).
func dirLike(abs string, d os.DirEntry) bool {
	if d.IsDir() {
		return true
	}
	if d.Type()&os.ModeSymlink != 0 {
		if info, err := os.Stat(abs); err == nil {
			return info.IsDir()
		}
	}
	return false
}

// hasSubdir reports whether a directory contains at least one subdirectory.
func hasSubdir(abs string) bool {
	entries, err := os.ReadDir(abs)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if dirLike(filepath.Join(abs, e.Name()), e) {
			return true
		}
	}
	return false
}

// handleFS implements GET /api/fs?path=<relative> — the folder list of the
// dialog for one directory.
func (s *Server) handleFS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rel := acl.CleanPath(r.URL.Query().Get("path"))
	abs, err := s.shareAbs(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dirents, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	entries := make([]fsEntry, 0, len(dirents))
	remoteRoots := s.remoteRoots()
	isRemote := func(abs string) bool {
		abs = filepath.Clean(abs)
		for _, r := range remoteRoots {
			if abs == r || strings.HasPrefix(abs, r+string(os.PathSeparator)) {
				return true
			}
		}
		return false
	}
	for _, d := range dirents {
		name := d.Name()
		full := filepath.Join(abs, name)
		e := fsEntry{Name: name, Path: joinRel(rel, name), Size: -1}
		if dirLike(full, d) {
			e.Dir = true
			e.Remote = isRemote(full)
			e.HasChildren = hasSubdir(full)
			if !e.Remote {
				e.Size = cachedSize(full)
			}
		} else if d.Type().IsRegular() {
			if info, statErr := d.Info(); statErr == nil {
				e.Size = info.Size()
			}
		} else {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	writeJSON(w, map[string]any{
		"path":    rel,
		"root":    s.shares.Root(),
		"share":   s.cfg.SMB.ShareName,
		"entries": entries,
	})
}

// handleFSSize implements GET /api/fs/size?path=<relative> — cached recursive
// size of one directory; the client polls until "ready" is true.
func (s *Server) handleFSSize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rel := acl.CleanPath(r.URL.Query().Get("path"))
	abs, err := s.shareAbs(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if info, statErr := os.Stat(abs); statErr != nil || !info.IsDir() {
		http.Error(w, "not a directory", http.StatusNotFound)
		return
	}
	if s.isRemotePath(abs) {
		// Recursive sizes over a 24 TB CIFS mount would hammer the Proxmox
		// host for hours; the dialog shows «—» for these disks instead.
		writeJSON(w, sizeState{Ready: true, Bytes: -1, Partial: true, At: time.Now()})
		return
	}
	writeJSON(w, requestSize(abs))
}

// handleFSMkdir implements POST /api/fs/mkdir {path, name} — «Создать папку в
// выделенной папке», exactly like the Keenetic dialog does.
func (s *Server) handleFSMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(payload.Name)
	if err := validEntryName(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parent, err := s.shareAbs(payload.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.Mkdir(filepath.Join(parent, name), 0o775); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{
		"status": "created",
		"path":   joinRel(acl.CleanPath(payload.Path), name),
	})
}

func validEntryName(name string) error {
	switch {
	case name == "":
		return os.ErrInvalid
	case name == "." || name == "..":
		return os.ErrInvalid
	case strings.ContainsAny(name, "/\\:*?\"<>|"):
		return os.ErrInvalid
	case strings.HasPrefix(name, ".") && len(name) <= 2:
		return os.ErrInvalid
	}
	for _, r := range name {
		if r < 32 {
			return os.ErrInvalid
		}
	}
	return nil
}

func joinRel(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}
