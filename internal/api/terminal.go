package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func (s *Server) handleTerminalSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions := s.termManager.ListSessions()
		s.sendJSON(w, http.StatusOK, sessions, "")

	case http.MethodPost:
		var req model.CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}

		sess, err := s.termManager.CreateSession(req.Target, req.Container, req.User, req.Title)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}

		s.sendJSON(w, http.StatusOK, model.TerminalSessionInfo{
			ID:            sess.ID,
			Title:         sess.Title,
			Target:        sess.Target,
			Container:     sess.Container,
			User:          sess.User,
			CreatedAt:     sess.CreatedAt.Unix(),
			ActiveClients: sess.ClientCount(),
			Cols:          sess.Cols,
			Rows:          sess.Rows,
			IsRunning:     sess.IsRunning(),
		}, "")

	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}

func (s *Server) handleTerminalSessionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected path: /api/terminal/sessions/{id} -> parts: ["api", "terminal", "sessions", "{id}"]
	if len(parts) < 4 {
		s.sendJSON(w, http.StatusBadRequest, nil, "session ID is required")
		return
	}
	sessionID := parts[3]

	if err := s.termManager.KillSession(sessionID); err != nil {
		s.sendJSON(w, http.StatusNotFound, nil, err.Error())
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "session terminated"}, "")
}
