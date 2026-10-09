package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthManager(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_auth_pass.txt")
	t.Setenv("DROIDSPACES_AUTH_FILE", tmpFile)

	// Test NewAuthManager constructor
	am := NewAuthManager()
	if am == nil {
		t.Fatalf("expected non-nil AuthManager")
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

	// Validate empty token and nonexistent token
	if am.ValidateToken("") {
		t.Errorf("expected empty token to be invalid")
	}
	if am.ValidateToken("nonexistent_token_123") {
		t.Errorf("expected nonexistent token to be invalid")
	}

	// Test AuthenticateRequest via header
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if !am.AuthenticateRequest(req) {
		t.Errorf("expected AuthenticateRequest via header to succeed")
	}

	// Test AuthenticateRequest via cookie
	reqCookie := httptest.NewRequest("GET", "/api/test", nil)
	reqCookie.AddCookie(&http.Cookie{Name: "ds_token", Value: token})
	if !am.AuthenticateRequest(reqCookie) {
		t.Errorf("expected AuthenticateRequest via cookie to succeed")
	}

	// Test AuthenticateRequest via query param
	reqQuery := httptest.NewRequest("GET", "/api/test?token="+token, nil)
	if !am.AuthenticateRequest(reqQuery) {
		t.Errorf("expected AuthenticateRequest via query param to succeed")
	}

	// Test ExtractToken without token
	reqEmpty := httptest.NewRequest("GET", "/api/test", nil)
	if tok := am.ExtractToken(reqEmpty); tok != "" {
		t.Errorf("expected empty token extracted, got %q", tok)
	}

	// Test ExtractToken with non-Bearer auth
	reqBasic := httptest.NewRequest("GET", "/api/test", nil)
	reqBasic.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	if tok := am.ExtractToken(reqBasic); tok != "" {
		t.Errorf("expected empty token for Basic auth, got %q", tok)
	}

	// Invalidate session
	am.InvalidateSession(token)
	if am.ValidateToken(token) {
		t.Errorf("expected invalidated token to be invalid")
	}

	// Test expired session cleanup
	expToken, _ := am.CreateSession()
	am.mu.Lock()
	sess := am.sessions[expToken]
	sess.ExpiresAt = time.Now().Add(-1 * time.Hour)
	am.sessions[expToken] = sess
	am.mu.Unlock()

	if am.ValidateToken(expToken) {
		t.Errorf("expected expired token to be invalid")
	}

	// Test ChangePassword: wrong old password
	if err := am.ChangePassword("WrongOldSecret", "NewSecret123"); err == nil {
		t.Errorf("expected error for wrong old password")
	}

	// Test ChangePassword: valid old password
	if err := am.ChangePassword("Droidspaces", "NewSecret123"); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	if !am.VerifyPassword("NewSecret123") {
		t.Errorf("expected NewSecret123 to match")
	}
	if am.VerifyPassword("Droidspaces") {
		t.Errorf("expected old password to fail")
	}

	// Test setPasswordInMemory
	am.setPasswordInMemory("DirectMemoryPassword")
	if !am.VerifyPassword("DirectMemoryPassword") {
		t.Errorf("expected direct memory password to match")
	}
}

func TestLoadExistingAndCorruptPasswordFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Corrupt password file (< 2 parts)
	corruptFile := filepath.Join(tmpDir, "corrupt.txt")
	_ = os.WriteFile(corruptFile, []byte("only_one_line_without_colon\n"), 0644)
	amCorrupt := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: corruptFile,
	}
	if err := amCorrupt.loadOrCreatePassword(); err != nil {
		t.Fatalf("loadOrCreatePassword on corrupt file failed: %v", err)
	}
	if !amCorrupt.VerifyPassword("Droidspaces") {
		t.Errorf("expected corrupt file to reset to default password Droidspaces")
	}

	// 2. Load existing valid password file
	validFile := filepath.Join(tmpDir, "valid.txt")
	am1 := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: validFile,
	}
	_ = am1.loadOrCreatePassword()
	_ = am1.ChangePassword("Droidspaces", "KeepAcrossRestarts")

	// Create second manager loading that file
	am2 := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: validFile,
	}
	if err := am2.loadOrCreatePassword(); err != nil {
		t.Fatalf("failed loading existing valid password: %v", err)
	}
	if !am2.VerifyPassword("KeepAcrossRestarts") {
		t.Errorf("expected existing password KeepAcrossRestarts to verify")
	}
}

func TestFindPasswordFilePathFallback(t *testing.T) {
	t.Setenv("DROIDSPACES_AUTH_FILE", "")
	path := findPasswordFilePath()
	if path == "" {
		t.Errorf("expected non-empty fallback password path")
	}
}
