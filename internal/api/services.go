package api

import (
	"encoding/json"
	"net/http"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func (s *Server) handleServicesRoute(w http.ResponseWriter, r *http.Request, containerName string, parts []string) {
	// GET /api/containers/{name}/services/{service}/logs
	if len(parts) == 6 && parts[5] == "logs" && r.Method == http.MethodGet {
		serviceName := parts[4]
		logs, err := s.client.GetServiceJournal(containerName, serviceName, 100)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"logs": logs}, "")
		return
	}

	// GET/POST/PUT/DELETE /api/containers/{name}/services/{service}/override
	if len(parts) == 6 && parts[5] == "override" {
		serviceName := parts[4]
		switch r.Method {
		case http.MethodGet:
			content, exists, err := s.client.GetSystemdOverride(containerName, serviceName)
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]interface{}{
				"content": content,
				"exists":  exists,
			}, "")
			return

		case http.MethodPost, http.MethodPut:
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.sendJSON(w, http.StatusBadRequest, nil, "invalid request body")
				return
			}
			if err := s.client.SetSystemdOverride(containerName, serviceName, req.Content); err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "override saved and systemd reloaded"}, "")
			return

		case http.MethodDelete:
			if err := s.client.DeleteSystemdOverride(containerName, serviceName); err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "override deleted and systemd reloaded"}, "")
			return

		default:
			s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
			return
		}
	}

	// GET /api/containers/{name}/services
	if r.Method == http.MethodGet {
		services, initSys, err := s.client.ListServices(containerName)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"services":    services,
			"init_system": initSys,
		}, "")
		return
	}

	// POST /api/containers/{name}/services
	if r.Method == http.MethodPost {
		var req model.ServiceActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if err := s.client.ManageService(containerName, req.Action, req.Service); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "service updated"}, "")
		return
	}

	s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
}
