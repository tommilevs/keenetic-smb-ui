package web

import (
	"net/http"

	"github.com/tommilevs/keenetic-smb-ui/internal/ndms"
)

// handleNDMSUsers exposes the router user list so the operator can import
// accounts into Samba with one click.
func (s *Server) handleNDMSUsers(w http.ResponseWriter, r *http.Request) {
	if !ndms.Available() {
		writeJSON(w, map[string]any{"available": false, "users": []string{}})
		return
	}
	users, err := ndms.ListUsers()
	if err != nil {
		writeJSON(w, map[string]any{"available": true, "error": err.Error()})
		return
	}
	known := make(map[string]bool, len(s.cfg.SMB.Users))
	for _, u := range s.cfg.SMB.Users {
		known[u.Name] = true
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"name":     u.Name,
			"readonly": u.Readonly,
			"manager":  u.Manager,
			"imported": known[u.Name],
		})
	}
	writeJSON(w, map[string]any{"available": true, "users": out})
}
