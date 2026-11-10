package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const sessionCookie = "ksmbui_session"

// sessions keeps short-lived session tokens in memory (restart resets them).
type sessions struct {
	mu   sync.Mutex
	data map[string]time.Time
	ttl  time.Duration
}

func newSessions() *sessions {
	return &sessions{data: make(map[string]time.Time), ttl: 12 * time.Hour}
}

func (s *sessions) create() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	token := hex.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.data[token] = time.Now().Add(s.ttl)
	return token
}

func (s *sessions) valid(token string) bool {
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.data[token]
	if !ok || time.Now().After(exp) {
		delete(s.data, token)
		return false
	}
	return true
}

func (s *sessions) drop(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, token)
}

func (s *sessions) gcLocked() {
	now := time.Now()
	for t, exp := range s.data {
		if now.After(exp) {
			delete(s.data, t)
		}
	}
}

// checkCredentials compares a login/password pair with the configured ones.
func (s *Server) checkCredentials(user, pass string) bool {
	return subtle.ConstantTimeCompare([]byte(user), []byte(s.cfg.Web.User)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(s.cfg.Web.Password)) == 1
}

// authenticated reports whether the request carries a valid session cookie or
// (for scripts and legacy clients) valid HTTP basic credentials.
func (s *Server) authenticated(r *http.Request) bool {
	if c, err := r.Cookie(sessionCookie); err == nil && s.sessions.valid(c.Value) {
		return true
	}
	user, pass, ok := r.BasicAuth()
	return ok && s.checkCredentials(user, pass)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.checkCredentials(body.User, body.Pass) {
		writeJSON(w, map[string]string{"status": "denied"})
		return
	}
	token := s.sessions.create()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.sessions.ttl.Seconds()),
	})
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.sessions.drop(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user := ""
	if s.authenticated(r) {
		user = s.cfg.Web.User
	}
	writeJSON(w, map[string]any{"authenticated": user != "", "user": user, "version": Version})
}
