package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthEndpointsAndMiddleware(t *testing.T) {
	// Clean up any test password file
	defer os.Remove(".webui_auth")

	server := NewServer(84)
	handler := server.Handler()

	// 1. Unauthenticated request to /api/status should return 401
	reqUnauth := httptest.NewRequest("GET", "/api/status", nil)
	rrUnauth := httptest.NewRecorder()
	handler.ServeHTTP(rrUnauth, reqUnauth)

	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for unauthenticated request, got %d", rrUnauth.Code)
	}

	// 2. Auth status endpoint should return authenticated=false
	reqStatus := httptest.NewRequest("GET", "/api/auth/status", nil)
	rrStatus := httptest.NewRecorder()
	handler.ServeHTTP(rrStatus, reqStatus)

	if rrStatus.Code != http.StatusOK {
		t.Errorf("expected status 200 for /api/auth/status, got %d", rrStatus.Code)
	}
	var statusBody map[string]interface{}
	_ = json.NewDecoder(rrStatus.Body).Decode(&statusBody)
	if data, ok := statusBody["data"].(map[string]interface{}); !ok || data["authenticated"] != false {
		t.Errorf("expected authenticated=false in /api/auth/status, got %v", statusBody)
	}

	// 3. Failed login
	bodyWrong, _ := json.Marshal(map[string]string{"password": "WrongPassword"})
	reqWrong := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(bodyWrong))
	rrWrong := httptest.NewRecorder()
	handler.ServeHTTP(rrWrong, reqWrong)

	if rrWrong.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for wrong password, got %d", rrWrong.Code)
	}

	// 4. Successful login with default password "Droidspaces"
	bodyCorrect, _ := json.Marshal(map[string]string{"password": "Droidspaces"})
	reqLogin := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(bodyCorrect))
	rrLogin := httptest.NewRecorder()
	handler.ServeHTTP(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		t.Fatalf("expected status 200 for login, got %d (%s)", rrLogin.Code, rrLogin.Body.String())
	}

	cookies := rrLogin.Result().Cookies()
	var dsCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "ds_token" {
			dsCookie = c
			break
		}
	}
	if dsCookie == nil || dsCookie.Value == "" {
		t.Fatalf("expected ds_token cookie to be set")
	}

	// 5. Authenticated request using cookie to /api/auth/status should return authenticated=true
	reqAuthStatus := httptest.NewRequest("GET", "/api/auth/status", nil)
	reqAuthStatus.AddCookie(dsCookie)
	rrAuthStatus := httptest.NewRecorder()
	handler.ServeHTTP(rrAuthStatus, reqAuthStatus)

	if rrAuthStatus.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rrAuthStatus.Code)
	}
	var authStatusBody map[string]interface{}
	_ = json.NewDecoder(rrAuthStatus.Body).Decode(&authStatusBody)
	if data, ok := authStatusBody["data"].(map[string]interface{}); !ok || data["authenticated"] != true {
		t.Errorf("expected authenticated=true, got %v", authStatusBody)
	}

	// 6. Logout
	reqLogout := httptest.NewRequest("POST", "/api/auth/logout", nil)
	reqLogout.AddCookie(dsCookie)
	rrLogout := httptest.NewRecorder()
	handler.ServeHTTP(rrLogout, reqLogout)

	if rrLogout.Code != http.StatusOK {
		t.Errorf("expected status 200 for logout, got %d", rrLogout.Code)
	}

	// 7. Re-check /api/auth/status after logout
	reqAfterLogout := httptest.NewRequest("GET", "/api/auth/status", nil)
	reqAfterLogout.AddCookie(dsCookie)
	rrAfterLogout := httptest.NewRecorder()
	handler.ServeHTTP(rrAfterLogout, reqAfterLogout)

	var afterLogoutBody map[string]interface{}
	_ = json.NewDecoder(rrAfterLogout.Body).Decode(&afterLogoutBody)
	if data, ok := afterLogoutBody["data"].(map[string]interface{}); !ok || data["authenticated"] != false {
		t.Errorf("expected authenticated=false after logout, got %v", afterLogoutBody)
	}
}
