package api

import (
	"encoding/json"
	"net/http"

	"github.com/latifangren/droidspaces-webui/internal/templates"
)

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	list := templates.ListTemplates()
	activeJob, progress, lastErr := templates.GetDownloadStatus()
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"templates":   list,
		"active_job":  activeJob,
		"progress":    progress,
		"last_error":  lastErr,
		"storage_dir": templates.GetStorageDir(),
	}, "")
}

func (s *Server) handleTemplateDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "template ID required")
		return
	}
	if err := templates.StartDownload(req.ID); err != nil {
		s.sendJSON(w, http.StatusConflict, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"message": "download started"}, "")
}

func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "template ID required")
		return
	}
	if err := templates.DeleteTemplate(req.ID); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
}
