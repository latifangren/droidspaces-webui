package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPassword = "Droidspaces"
	SessionDuration = 7 * 24 * time.Hour
)

type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AuthManager struct {
	mu           sync.RWMutex
	passwordHash string
	salt         string
	sessions     map[string]Session
	pwFilePath   string
}

func NewAuthManager() *AuthManager {
	pwFile := findPasswordFilePath()
	am := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: pwFile,
	}

	if err := am.loadOrCreatePassword(); err != nil {
		log.Printf("[auth] warning: failed to load password file (%v), using default in memory", err)
		am.setPasswordInMemory(DefaultPassword)
	}

	return am
}

func findPasswordFilePath() string {
	if custom := os.Getenv("DROIDSPACES_AUTH_FILE"); custom != "" {
		return custom
	}
	candidateDirs := []string{
		"/data/local/Droidspaces",
		"/data/adb/modules/droidspaces-webui",
		"/tmp/droidspaces",
		".",
	}

	for _, dir := range candidateDirs {
		if _, err := os.Stat(dir); err == nil {
			return filepath.Join(dir, ".webui_auth")
		}
	}
	return ".webui_auth"
}

func hashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + ":" + password))
	return hex.EncodeToString(h.Sum(nil))
}

func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (am *AuthManager) setPasswordInMemory(password string) {
	salt, _ := generateRandomHex(16)
	am.mu.Lock()
	defer am.mu.Unlock()
	am.salt = salt
	am.passwordHash = hashPassword(password, salt)
}

func (am *AuthManager) loadOrCreatePassword() error {
	data, err := os.ReadFile(am.pwFilePath)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) >= 2 {
			am.mu.Lock()
			am.salt = strings.TrimSpace(lines[0])
			am.passwordHash = strings.TrimSpace(lines[1])
			am.mu.Unlock()
			return nil
		}
	}

	// Password file not found or invalid format: initialize with default password
	salt, err := generateRandomHex(16)
	if err != nil {
		return err
	}
	hash := hashPassword(DefaultPassword, salt)

	_ = os.MkdirAll(filepath.Dir(am.pwFilePath), 0755)
	content := fmt.Sprintf("%s\n%s\n", salt, hash)
	if err := os.WriteFile(am.pwFilePath, []byte(content), 0600); err != nil {
		log.Printf("[auth] could not write password file %s: %v", am.pwFilePath, err)
	}

	am.mu.Lock()
	am.salt = salt
	am.passwordHash = hash
	am.mu.Unlock()
	return nil
}

func (am *AuthManager) VerifyPassword(password string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	calculated := hashPassword(password, am.salt)
	return subtle.ConstantTimeCompare([]byte(calculated), []byte(am.passwordHash)) == 1
}

func (am *AuthManager) ChangePassword(oldPassword, newPassword string) error {
	if !am.VerifyPassword(oldPassword) {
		return fmt.Errorf("current password incorrect")
	}

	if len(strings.TrimSpace(newPassword)) < 4 {
		return fmt.Errorf("new password must be at least 4 characters")
	}

	salt, err := generateRandomHex(16)
	if err != nil {
		return err
	}
	hash := hashPassword(newPassword, salt)

	content := fmt.Sprintf("%s\n%s\n", salt, hash)
	if err := os.WriteFile(am.pwFilePath, []byte(content), 0600); err != nil {
		log.Printf("[auth] failed to persist new password: %v", err)
	}

	am.mu.Lock()
	am.salt = salt
	am.passwordHash = hash
	// Invalidate all existing sessions on password change
	am.sessions = make(map[string]Session)
	am.mu.Unlock()

	return nil
}

func (am *AuthManager) CreateSession() (string, error) {
	token, err := generateRandomHex(32)
	if err != nil {
		return "", err
	}

	now := time.Now()
	am.mu.Lock()
	defer am.mu.Unlock()

	am.sessions[token] = Session{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionDuration),
	}

	return token, nil
}

func (am *AuthManager) InvalidateSession(token string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.sessions, token)
}

func (am *AuthManager) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	am.mu.RLock()
	sess, exists := am.sessions[token]
	am.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(sess.ExpiresAt) {
		am.mu.Lock()
		delete(am.sessions, token)
		am.mu.Unlock()
		return false
	}

	return true
}

func (am *AuthManager) ExtractToken(r *http.Request) string {
	// 1. Check Authorization: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	// 2. Check Cookie "ds_token"
	if cookie, err := r.Cookie("ds_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. Check Query parameter "token" (standard for WebSocket handshakes)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}

func (am *AuthManager) AuthenticateRequest(r *http.Request) bool {
	token := am.ExtractToken(r)
	return am.ValidateToken(token)
}
