package web

import (
	"net/http"

	"github.com/tommilevs/keenetic-smb-ui/internal/update"
)

// handleUpdate checks GitHub for a newer release (GET) or installs it (POST).
func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	arch := update.DetectArch()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, update.Check(Version, arch))
	case http.MethodPost:
		info := update.Check(Version, arch)
		if info.Error != "" {
			writeJSON(w, map[string]any{"status": "error", "error": info.Error, "info": info})
			return
		}
		if !info.Newer || info.URL == "" {
			writeJSON(w, map[string]any{"status": "uptodate", "info": info})
			return
		}
		if err := update.Install(info.URL); err != nil {
			writeJSON(w, map[string]any{"status": "error", "error": err.Error(), "info": info})
			return
		}
		writeJSON(w, map[string]any{
			"status": "installing",
			"info":   info,
			"hint":   "пакет скачивается и устанавливается, служба перезапустится через несколько секунд",
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
