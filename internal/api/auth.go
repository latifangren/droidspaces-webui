package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type loginRequest struct {
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	authenticated := s.authMgr.AuthenticateRequest(r)
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": authenticated,
	}, "")
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid request body")
		return
	}

	if !s.authMgr.VerifyPassword(req.Password) {
		s.sendJSON(w, http.StatusUnauthorized, nil, "password incorrect")
		return
	}

	token, err := s.authMgr.CreateSession()
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, "failed to create session")
		return
	}

	// Set secure HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "ds_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600, // 7 days
	})

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"token":   token,
		"message": "login successful",
	}, "")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	token := s.authMgr.ExtractToken(r)
	if token != "" {
		s.authMgr.InvalidateSession(token)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "ds_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "logged out"}, "")
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid request body")
		return
	}

	if strings.TrimSpace(req.NewPassword) == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "new password cannot be empty")
		return
	}

	if err := s.authMgr.ChangePassword(req.CurrentPassword, req.NewPassword); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	// Create new session for caller after password change
	newToken, err := s.authMgr.CreateSession()
	if err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "ds_token",
			Value:    newToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600,
		})
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"token":   newToken,
		"message": "password updated successfully",
	}, "")
}
