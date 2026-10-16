package auth

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPBKDF2SHA256KnownVector(t *testing.T) {
	got := hex.EncodeToString(pbkdf2SHA256([]byte("password"), []byte("salt"), 1, 32))
	want := "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"
	if got != want {
		t.Fatalf("PBKDF2-SHA256 vector mismatch: got %s, want %s", got, want)
	}
}

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

	// Verify the persisted credential remains available after password change.
	amReloaded := &AuthManager{sessions: make(map[string]Session), pwFilePath: tmpFile}
	if err := amReloaded.loadOrCreatePassword(); err != nil || !amReloaded.VerifyPassword("NewSecret123") {
		t.Errorf("expected changed password to persist across restart: %v", err)
	}
}

func TestLoadExistingAndCorruptPasswordFile(t *testing.T) {
	tmpDir := t.TempDir()

	// A corrupt password file must fail closed rather than reset to a known password.
	corruptFile := filepath.Join(tmpDir, "corrupt.txt")
	_ = os.WriteFile(corruptFile, []byte("only_one_line_without_colon\n"), 0644)
	amCorrupt := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: corruptFile,
	}
	if err := amCorrupt.loadOrCreatePassword(); err == nil {
		t.Fatalf("expected corrupt password file to fail closed")
	}
	if amCorrupt.VerifyPassword("Droidspaces") {
		t.Errorf("corrupt file must not enable the default password")
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

func TestLegacyPasswordMigratesOnSuccessfulVerification(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "auth")
	salt := "00112233445566778899aabbccddeeff"
	legacyHash := legacyHashPassword("old-password", salt)
	if err := os.WriteFile(tmpFile, []byte(salt+"\n"+legacyHash+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	am := &AuthManager{sessions: make(map[string]Session), pwFilePath: tmpFile}
	if err := am.loadOrCreatePassword(); err != nil {
		t.Fatal(err)
	}
	if !am.VerifyPassword("old-password") {
		t.Fatal("legacy password did not verify")
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), passwordHashPrefix+"$") {
		t.Fatalf("legacy password record was not upgraded: %q", data)
	}
	if !am.VerifyPassword("old-password") {
		t.Fatal("migrated password did not verify")
	}
}

func TestPasswordChangePersistenceFailureKeepsOldCredential(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "auth")
	am := &AuthManager{sessions: make(map[string]Session), pwFilePath: file}
	if err := am.loadOrCreatePassword(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := am.ChangePassword("Droidspaces", "replacement-secret"); err == nil {
		t.Fatal("expected persistence failure")
	}
	if !am.VerifyPassword("Droidspaces") || am.VerifyPassword("replacement-secret") {
		t.Fatal("credential changed despite persistence failure")
	}
}

func TestFindPasswordFilePathFallback(t *testing.T) {
	t.Setenv("DROIDSPACES_AUTH_FILE", "")
	path := findPasswordFilePath()
	if path == "" {
		t.Errorf("expected non-empty fallback password path")
	}
}
