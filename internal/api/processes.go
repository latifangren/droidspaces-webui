package api

import (
	"encoding/json"
	"net/http"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func (s *Server) handleProcessesRoute(w http.ResponseWriter, r *http.Request, containerName string, parts []string) {
	// POST /api/containers/{name}/processes/kill
	if len(parts) >= 5 && parts[4] == "kill" && r.Method == http.MethodPost {
		var req model.KillProcessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if err := s.client.KillProcess(containerName, req.PID, req.Signal); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "process killed"}, "")
		return
	}

	// GET /api/containers/{name}/processes
	if r.Method == http.MethodGet {
		procs, err := s.client.ListProcesses(containerName)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]interface{}{"processes": procs}, "")
		return
	}

	s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
}

func (s *Server) handleUsersRoute(w http.ResponseWriter, r *http.Request, containerName string) {
	users, err := s.client.ListUsers(containerName)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]interface{}{"users": users}, "")
}
