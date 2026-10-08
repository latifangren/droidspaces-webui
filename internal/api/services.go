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
