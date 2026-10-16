package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	mu                   sync.RWMutex
	passwordHash         string
	salt                 string
	iterations           int
	sessions             map[string]Session
	pwFilePath           string
	usingDefaultPassword bool
}

func NewAuthManager() *AuthManager {
	pwFile := findPasswordFilePath()
	am := &AuthManager{
		sessions:   make(map[string]Session),
		pwFilePath: pwFile,
	}

	if err := am.loadOrCreatePassword(); err != nil {
		log.Printf("[auth] error: password authentication unavailable: %v", err)
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

const (
	passwordHashIterations = 120_000
	passwordHashKeyLength  = 32
	passwordHashPrefix     = "pbkdf2-sha256"
)

func hashPassword(password, salt string) string {
	return hashPasswordWithIterations(password, salt, passwordHashIterations)
}

func hashPasswordWithIterations(password, salt string, iterations int) string {
	return hex.EncodeToString(pbkdf2SHA256([]byte(password), []byte(salt), iterations, passwordHashKeyLength))
}

func legacyHashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + ":" + password))
	return hex.EncodeToString(h.Sum(nil))
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hLen := sha256.Size
	blocks := (keyLen + hLen - 1) / hLen
	derived := make([]byte, 0, blocks*hLen)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		value := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range value {
				value[j] ^= u[j]
			}
		}
		derived = append(derived, value...)
	}
	return derived[:keyLen]
}

func encodePasswordRecord(salt, hash string) []byte {
	return []byte(fmt.Sprintf("%s$%d$%s\n%s\n", passwordHashPrefix, passwordHashIterations, salt, hash))
}

func (am *AuthManager) persistPasswordRecord(salt, hash string) error {
	dir := filepath.Dir(am.pwFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".webui_auth-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(encodePasswordRecord(salt, hash)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, am.pwFilePath)
}

func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (am *AuthManager) loadOrCreatePassword() error {
	data, err := os.ReadFile(am.pwFilePath)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) < 2 {
			return fmt.Errorf("invalid password file format; remove or restore %s to recover access", am.pwFilePath)
		}
		first := strings.TrimSpace(lines[0])
		hash := strings.TrimSpace(lines[1])
		salt := first
		if strings.HasPrefix(first, passwordHashPrefix+"$") {
			parts := strings.Split(first, "$")
			if len(parts) != 3 {
				return fmt.Errorf("invalid password hash header")
			}
			iterations, parseErr := strconv.Atoi(parts[1])
			if parseErr != nil || iterations < 10_000 || iterations > 1_000_000 {
				return fmt.Errorf("invalid password hash iteration count")
			}
			if len(hash) != passwordHashKeyLength*2 {
				return fmt.Errorf("invalid password hash length")
			}
			if _, decodeErr := hex.DecodeString(hash); decodeErr != nil {
				return fmt.Errorf("invalid password hash encoding")
			}
			salt = parts[2]
			am.mu.Lock()
			am.salt = salt
			am.passwordHash = hash
			am.iterations = iterations
			am.mu.Unlock()
			am.usingDefaultPassword = subtle.ConstantTimeCompare([]byte(hashPasswordWithIterations(DefaultPassword, salt, iterations)), []byte(hash)) == 1
			return nil
		}
		if len(hash) != sha256.Size*2 {
			return fmt.Errorf("invalid legacy password hash length")
		}
		if _, decodeErr := hex.DecodeString(hash); decodeErr != nil {
			return fmt.Errorf("invalid legacy password hash encoding")
		}
		am.mu.Lock()
		am.salt = salt
		am.passwordHash = hash
		am.iterations = 0
		am.mu.Unlock()
		am.usingDefaultPassword = subtle.ConstantTimeCompare([]byte(hash), []byte(legacyHashPassword(DefaultPassword, salt))) == 1
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("read password file: %w", err)
	}

	salt, err := generateRandomHex(16)
	if err != nil {
		return err
	}
	hash := hashPassword(DefaultPassword, salt)
	if err := am.persistPasswordRecord(salt, hash); err != nil {
		return fmt.Errorf("persist initial password: %w", err)
	}
	am.mu.Lock()
	am.salt = salt
	am.passwordHash = hash
	am.iterations = passwordHashIterations
	am.usingDefaultPassword = true
	am.mu.Unlock()
	return nil
}

func (am *AuthManager) VerifyPassword(password string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	legacy := am.iterations == 0
	calculated := legacyHashPassword(password, am.salt)
	if !legacy {
		calculated = hashPasswordWithIterations(password, am.salt, am.iterations)
	}
	if subtle.ConstantTimeCompare([]byte(calculated), []byte(am.passwordHash)) != 1 {
		return false
	}
	if legacy {
		am.usingDefaultPassword = password == DefaultPassword
		salt, err := generateRandomHex(16)
		if err != nil {
			return true
		}
		hash := hashPassword(password, salt)
		if err := am.persistPasswordRecord(salt, hash); err == nil {
			am.salt = salt
			am.passwordHash = hash
			am.iterations = passwordHashIterations
		}
	}
	return true
}

func (am *AuthManager) IsUsingDefaultPassword() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.usingDefaultPassword
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
	if err := am.persistPasswordRecord(salt, hash); err != nil {
		return fmt.Errorf("failed to persist new password: %w", err)
	}

	am.mu.Lock()
	am.salt = salt
	am.passwordHash = hash
	am.iterations = passwordHashIterations
	// Invalidate all existing sessions on password change
	am.usingDefaultPassword = false
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
