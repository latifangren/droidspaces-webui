package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthManager(t *testing.T) {
	tmpFile := "test_auth_pass.txt"
	defer os.Remove(tmpFile)

	am := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: tmpFile,
	}
	if err := am.loadOrCreatePassword(); err != nil {
		t.Fatalf("loadOrCreatePassword failed: %v", err)
	}

	// Verify default password
	if !am.VerifyPassword("Droidspaces") {
		t.Errorf("expected default password Droidspaces to match")
	}
	if am.VerifyPassword("wrongpass") {
		t.Errorf("expected wrong password to fail")
	}

	// Create session
	token, err := am.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if !am.ValidateToken(token) {
		t.Errorf("expected token to be valid")
	}

	// Test request authentication via header
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if !am.AuthenticateRequest(req) {
		t.Errorf("expected AuthenticateRequest via header to succeed")
	}

	// Test request authentication via cookie
	reqCookie := httptest.NewRequest("GET", "/api/status", nil)
	reqCookie.AddCookie(&http.Cookie{Name: "ds_token", Value: token})
	if !am.AuthenticateRequest(reqCookie) {
		t.Errorf("expected AuthenticateRequest via cookie to succeed")
	}

	// Test request authentication via query param
	reqQuery := httptest.NewRequest("GET", "/api/ws/terminal?token="+token, nil)
	if !am.AuthenticateRequest(reqQuery) {
		t.Errorf("expected AuthenticateRequest via query param to succeed")
	}

	// Test change password
	err = am.ChangePassword("Droidspaces", "NewSecret123")
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	if !am.VerifyPassword("NewSecret123") {
		t.Errorf("expected NewSecret123 to match")
	}
	if am.VerifyPassword("Droidspaces") {
		t.Errorf("expected old password to fail")
	}
	// Old sessions should be invalidated
	if am.ValidateToken(token) {
		t.Errorf("expected old session token to be invalidated after password change")
	}
}
