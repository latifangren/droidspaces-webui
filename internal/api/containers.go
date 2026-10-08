package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		res, err := s.client.Show()
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, res, "")

	case http.MethodPost:
		var req model.StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if req.Name == "" {
			s.sendJSON(w, http.StatusBadRequest, nil, "container name is required")
			return
		}
		if err := s.client.Start(req); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "container created and started"}, "")

	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}

func (s *Server) handleContainerAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid container url")
		return
	}
	name := parts[2]

	// GET /api/containers/{name}
	if len(parts) == 3 && r.Method == http.MethodGet {
		info, err := s.client.Info(name)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, info, "")
		return
	}

	// DELETE /api/containers/{name}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		if err := s.client.Delete(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
		return
	}

	if len(parts) >= 4 {
		action := parts[3]

		// Init System Services
		if action == "services" {
			s.handleServicesRoute(w, r, name, parts)
			return
		}

		// Container Processes
		if action == "processes" {
			s.handleProcessesRoute(w, r, name, parts)
			return
		}

		// Container Users
		if action == "users" && r.Method == http.MethodGet {
			s.handleUsersRoute(w, r, name)
			return
		}

		// Lifecycle Actions
		if r.Method == http.MethodPost {
			s.handleLifecycleRoute(w, r, name, action)
			return
		}
	}

	s.sendJSON(w, http.StatusNotFound, nil, "endpoint not found")
}

func (s *Server) handleLifecycleRoute(w http.ResponseWriter, r *http.Request, name, action string) {
	switch action {
	case "start":
		if err := s.client.Start(model.StartRequest{Name: name}); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "started"}, "")
	case "stop":
		if err := s.client.Stop(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "stopped"}, "")
	case "restart":
		if err := s.client.Restart(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "restarted"}, "")
	case "delete":
		if err := s.client.Delete(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
	case "exec":
		var req model.ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		out, err := s.client.Exec(name, req.Command, req.User)
		if err != nil {
			s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 1}, "")
			return
		}
		s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 0}, "")
	default:
		s.sendJSON(w, http.StatusBadRequest, nil, "unknown action: "+action)
	}
}
