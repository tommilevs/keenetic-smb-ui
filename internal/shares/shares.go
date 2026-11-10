// Package shares mounts external SMB shares (Proxmox hosts) and builds the
// aggregated directory tree that the managed Samba instance exports.
package shares

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tommilevs/keenetic-smb-ui/internal/config"
)

// Manager reconciles configured remotes with the running mounts.
type Manager struct {
	mu  sync.Mutex
	cfg *config.Config
}

// New returns a manager bound to the live configuration.
func New(cfg *config.Config) *Manager { return &Manager{cfg: cfg} }

// Root returns the aggregated share root.
func (m *Manager) Root() string {
	root := m.cfg.SMB.SharePath
	if root == "" {
		root = "/opt/var/keenetic-smb-ui/share"
	}
	return root
}

// MountPoint returns where a remote is mounted locally.
func (m *Manager) MountPoint(r config.Remote) string {
	return filepath.Join(m.Root(), sanitize(r.Name))
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "remote"
	}
	repl := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_", "..", "_")
	return repl.Replace(s)
}

// Apply ensures the aggregate root, symlinks to local disks and all enabled
// remotes are mounted. Disabled or removed remotes are unmounted.
func (m *Manager) Apply() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	if err := ensureCIFS(); err != nil {
		errs = append(errs, err)
	}
	root := m.Root()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return []error{err}
	}

	// symlink local router disks (/tmp/mnt/*) as top-level folders
	if entries, err := os.ReadDir("/tmp/mnt"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			src := filepath.Join("/tmp/mnt", e.Name())
			dst := filepath.Join(root, "local-"+e.Name())
			if _, err := os.Lstat(dst); os.IsNotExist(err) {
				if err := os.Symlink(src, dst); err != nil {
					errs = append(errs, fmt.Errorf("symlink %s: %w", dst, err))
				}
			}
		}
	}

	for _, r := range m.cfg.Remotes {
		mp := m.MountPoint(r)
		mounted := isMounted(mp)
		switch {
		case r.Enabled && !mounted:
			if err := os.MkdirAll(mp, 0o755); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := mountRemote(r, mp); err != nil {
				errs = append(errs, err)
			}
		case !r.Enabled && mounted:
			if err := exec.Command("umount", mp).Run(); err != nil {
				errs = append(errs, fmt.Errorf("umount %s: %w", mp, err))
			}
		}
	}
	return errs
}

// UnmountAll unmounts every configured remote (used on shutdown).
func (m *Manager) UnmountAll() {
	for _, r := range m.cfg.Remotes {
		mp := m.MountPoint(r)
		if isMounted(mp) {
			_ = exec.Command("umount", mp).Run()
		}
	}
}

// Status reports mount state per remote for the UI.
type Status struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Target  string `json:"target"`
	Mounted bool   `json:"mounted"`
	Error   string `json:"error,omitempty"`
}

// Statuses returns the current mount state of all remotes.
func (m *Manager) Statuses() []Status {
	out := make([]Status, 0, len(m.cfg.Remotes))
	for _, r := range m.cfg.Remotes {
		mp := m.MountPoint(r)
		st := Status{ID: r.ID, Name: r.Name, Target: mp, Mounted: isMounted(mp)}
		if !st.Mounted && r.Enabled {
			st.Error = "not mounted"
		}
		out = append(out, st)
	}
	return out
}

func mountRemote(r config.Remote, mp string) error {
	// Keenetic kernels (4.9-ndm) ship a cifs client limited to SMB 2.0/2.1,
	// so 2.1 is the default; Options can override it explicitly.
	opts := []string{
		"vers=2.1", "username=" + r.User, "password=" + r.Password,
		"uid=0", "gid=0", "file_mode=0775", "dir_mode=0775", "iocharset=utf8",
		"nounix", "serverino", "actimeo=10", "nomapposix",
	}
	if r.ReadOnly {
		opts = append(opts, "ro")
	}
	if extra := strings.TrimSpace(r.Options); extra != "" {
		parts := strings.Split(extra, ",")
		for _, p := range parts {
			if strings.HasPrefix(p, "vers=") {
				opts[0] = p
				continue
			}
			opts = append(opts, p)
		}
	}
	target := "//" + r.Host + "/" + strings.TrimPrefix(r.Share, "/")
	if r.SubPath != "" {
		target += "/" + strings.TrimPrefix(r.SubPath, "/")
	}
	return platformMount(target, mp, strings.Join(opts, ","))
}

// isMounted parses /proc/mounts to avoid depending on mountpoint(1).
func isMounted(path string) bool {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && unescape(fields[1]) == path {
			return true
		}
	}
	return false
}

func unescape(s string) string {
	return strings.NewReplacer("\\040", " ", "\\011", "\t", "\\012", "\n").Replace(s)
}
