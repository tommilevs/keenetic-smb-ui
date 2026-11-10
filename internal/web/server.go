// Package web serves the embedded web UI and its JSON API.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tommilevs/keenetic-smb-ui/internal/config"
	"github.com/tommilevs/keenetic-smb-ui/internal/samba"
	"github.com/tommilevs/keenetic-smb-ui/internal/shares"
	"github.com/tommilevs/keenetic-smb-ui/internal/stats"
)

//go:embed assets
var assets embed.FS

// Server bundles HTTP handlers with the domain managers.
type Server struct {
	cfg      *config.Config
	smb      *samba.Manager
	shares   *shares.Manager
	stats    *stats.Collector
	sessions *sessions
	http     *http.Server
}

// New wires the server.
func New(cfg *config.Config) (*Server, error) {
	sh := shares.New(cfg)
	paths := []string{sh.Root()}
	for _, r := range cfg.Remotes {
		paths = append(paths, sh.MountPoint(r))
	}
	s := &Server{
		cfg:      cfg,
		smb:      samba.New(cfg),
		shares:   sh,
		stats:    stats.New(cfg.Stats.HistoryLen, paths),
		sessions: newSessions(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/status", s.auth(s.handleStatus))
	mux.HandleFunc("/api/config", s.auth(s.handleConfig))
	mux.HandleFunc("/api/config/raw", s.auth(s.handleRawConfig))
	mux.HandleFunc("/api/service", s.auth(s.handleService))
	mux.HandleFunc("/api/remotes", s.auth(s.handleRemotes))
	mux.HandleFunc("/api/remotes/apply", s.auth(s.handleApply))
	mux.HandleFunc("/api/remotes/status", s.auth(s.handleRemoteStatus))
	mux.HandleFunc("/api/users", s.auth(s.handleUsers))
	mux.HandleFunc("/api/logs", s.auth(s.handleLogs))
	mux.HandleFunc("/api/update", s.auth(s.handleUpdate))

	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	s.http = &http.Server{
		Addr:              cfg.Web.Listen,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	ln, err := net.Listen("tcp", s.cfg.Web.Listen)
	if err != nil {
		return err
	}
	log.Printf("web ui on %s", ln.Addr())
	err = s.http.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown stops the server gracefully.
func (s *Server) Shutdown(d time.Duration) error {
	ctx, cancel := contextWithTimeout(d)
	defer cancel()
	return s.http.Shutdown(ctx)
}

// Bootstrap brings the managed stack up after start: IP alias, smb.conf,
// users, smbd and the aggregated mounts. Failures are logged, never fatal —
// the UI must stay reachable so the operator can fix things.
func (s *Server) Bootstrap() {
	go func() {
		if !s.cfg.SMB.Enabled {
			return
		}
		if changed, err := s.smb.EnsureBindIP(); err != nil {
			log.Printf("bootstrap: bind ip: %v", err)
		} else if changed {
			log.Printf("bootstrap: assigned %s", s.cfg.SMB.ListenIP)
		}
		if err := s.smb.WriteConfig(); err != nil {
			log.Printf("bootstrap: write config: %v", err)
		}
		if err := s.smb.SyncUsers(); err != nil {
			log.Printf("bootstrap: users: %v", err)
		}
		if err := s.smb.Start(); err != nil {
			log.Printf("bootstrap: smbd: %v", err)
		}
		for _, e := range s.shares.Apply() {
			log.Printf("bootstrap: share: %v", e)
		}
		log.Printf("bootstrap: done, smb=%s", s.smb.Status())
	}()
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="keenetic-smb-ui"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if s.cfg.Web.ReadOnly && r.Method != http.MethodGet {
			http.Error(w, "read-only mode", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	snap := s.stats.Sample()
	writeJSON(w, map[string]any{
		"service":  s.smb.Status(),
		"smb_ip":   s.cfg.SMB.ListenIP,
		"share":    s.cfg.SMB.ShareName,
		"root":     s.shares.Root(),
		"stats":    snap,
		"remotes":  s.shares.Statuses(),
		"version":  Version,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		safe := *s.cfg
		for i := range safe.SMB.Users {
			safe.SMB.Users[i].Password = ""
		}
		for i := range safe.Remotes {
			safe.Remotes[i].Password = ""
		}
		writeJSON(w, &safe)
	case http.MethodPost:
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := s.cfg.Update(func(c *config.Config) error {
			incoming.Paths.ConfigFile = c.Paths.ConfigFile
			incoming.Paths.DataDir = c.Paths.DataDir
			if incoming.Paths.SambaConf == "" {
				incoming.Paths.SambaConf = c.Paths.SambaConf
			}
			if incoming.Paths.SambaPidDir == "" {
				incoming.Paths.SambaPidDir = c.Paths.SambaPidDir
			}
			*c = incoming
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "saved"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRawConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(s.smb.Config())); err != nil {
		log.Printf("raw config write: %v", err)
	}
}

func (s *Server) handleService(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	var err error
	msgs := map[string]string{}
	switch action {
	case "start":
		if changed, e := s.smb.EnsureBindIP(); e != nil {
			err = e
		} else if changed {
			msgs["ip"] = "assigned " + s.cfg.SMB.ListenIP
		}
		if err == nil {
			err = s.smb.WriteConfig()
		}
		if err == nil {
			err = s.smb.SyncUsers()
		}
		if err == nil {
			err = s.smb.Start()
		}
	case "stop":
		err = s.smb.Stop()
	case "restart":
		err = s.smb.WriteConfig()
		if err == nil {
			err = s.smb.SyncUsers()
		}
		if err == nil {
			_, _ = s.smb.EnsureBindIP()
			err = s.smb.Restart()
		}
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"status": "error", "error": err.Error(), "service": s.smb.Status()})
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "service": s.smb.Status(), "info": msgs})
}

func (s *Server) handleRemotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Remotes []config.Remote `json:"remotes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.cfg.Update(func(c *config.Config) error {
		// keep passwords of entries that did not send one
		old := map[string]string{}
		for _, r := range c.Remotes {
			old[r.ID] = r.Password
		}
		for i := range payload.Remotes {
			if payload.Remotes[i].Password == "" {
				payload.Remotes[i].Password = old[payload.Remotes[i].ID]
			}
			if payload.Remotes[i].ID == "" {
				payload.Remotes[i].ID = fmt.Sprintf("r%d", time.Now().UnixNano())
			}
			payload.Remotes[i].Name = strings.TrimSpace(payload.Remotes[i].Name)
		}
		c.Remotes = payload.Remotes
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "saved"})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	errs := s.shares.Apply()
	if err := s.smb.WriteConfig(); err != nil {
		errs = append(errs, err)
	}
	if err := s.smb.Restart(); err != nil {
		errs = append(errs, err)
	}
	paths := []string{s.shares.Root()}
	for _, rm := range s.cfg.Remotes {
		paths = append(paths, s.shares.MountPoint(rm))
	}
	s.stats.SetSharePaths(paths)
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	writeJSON(w, map[string]any{"status": "applied", "errors": msgs})
}

func (s *Server) handleRemoteStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.shares.Statuses())
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var users []config.User
	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.cfg.Update(func(c *config.Config) error {
		old := map[string]string{}
		for _, u := range c.SMB.Users {
			old[u.Name] = u.Password
		}
		for i := range users {
			if users[i].Password == "" {
				users[i].Password = old[users[i].Name]
			}
		}
		c.SMB.Users = users
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.smb.SyncUsers(); err != nil {
		writeJSON(w, map[string]any{"status": "saved", "warning": err.Error()})
		return
	}
	_ = s.smb.Restart()
	writeJSON(w, map[string]string{"status": "saved"})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	const logPath = "/opt/var/keenetic-smb-ui/log/samba.log"
	data, err := os.ReadFile(logPath)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "log not available: %v\n", err)
		return
	}
	const tail = 32768
	if len(data) > tail {
		data = data[len(data)-tail:]
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// The UI is embedded in the binary and ships with every release; never
		// let a browser serve a stale bundle after an update.
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
