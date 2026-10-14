package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultLoginRateLimit is the maximum number of login attempts allowed within the window.
	DefaultLoginRateLimit = 5
	// DefaultLoginRateWindow is the duration of the rate limit window.
	DefaultLoginRateWindow = 60 * time.Second
)

// rateLimiter implements an in-memory sliding-window rate limiter per client IP.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// newRateLimiter creates a new sliding-window rate limiter with custom limit and window.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// newLoginRateLimiter creates a rate limiter with default login limits (5 attempts per 60s).
func newLoginRateLimiter() *rateLimiter {
	return newRateLimiter(DefaultLoginRateLimit, DefaultLoginRateWindow)
}

// Allow records an attempt for the given IP and reports whether the attempt is permitted.
// If not allowed, it returns false and the retry-after duration until the oldest attempt expires.
func (rl *rateLimiter) Allow(ip string) (bool, time.Duration) {
	if rl == nil {
		return true, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Keep only attempts within window
	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.attempts[ip] = valid
		oldest := valid[0]
		retryAfter := rl.window - now.Sub(oldest)
		if retryAfter <= 0 {
			retryAfter = time.Second
		}
		return false, retryAfter
	}

	valid = append(valid, now)
	rl.attempts[ip] = valid

	// Periodic housekeeping: clean up stale IPs if map grows
	if len(rl.attempts) > 100 {
		for k, v := range rl.attempts {
			var active []time.Time
			for _, t := range v {
				if t.After(cutoff) {
					active = append(active, t)
				}
			}
			if len(active) == 0 {
				delete(rl.attempts, k)
			} else {
				rl.attempts[k] = active
			}
		}
	}

	return true, 0
}

// Reset clears recorded attempts for a specific IP.
func (rl *rateLimiter) Reset(ip string) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// ResetAll clears all recorded attempts.
func (rl *rateLimiter) ResetAll() {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts = make(map[string][]time.Time)
}

// getClientIP extracts the client IP address from the request.
// It checks standard proxy headers (X-Forwarded-For, X-Real-IP)
// before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
