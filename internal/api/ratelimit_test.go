package api

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := newRateLimiter(5, 60*time.Second)
	ip := "192.168.1.100"

	// First 5 attempts must succeed
	for i := range 5 {
		allowed, retryAfter := rl.Allow(ip)
		if !allowed {
			t.Fatalf("attempt %d should be allowed, got blocked", i+1)
		}
		if retryAfter != 0 {
			t.Fatalf("attempt %d retryAfter should be 0, got %v", i+1, retryAfter)
		}
	}

	// 6th attempt must be rejected
	allowed, retryAfter := rl.Allow(ip)
	if allowed {
		t.Fatalf("attempt 6 should be blocked")
	}
	if retryAfter <= 0 || retryAfter > 60*time.Second {
		t.Fatalf("expected retryAfter between 0 and 60s, got %v", retryAfter)
	}

	// Reset should restore attempts
	rl.Reset(ip)
	allowed, _ = rl.Allow(ip)
	if !allowed {
		t.Fatalf("attempt after reset should be allowed")
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	rl := newRateLimiter(5, 60*time.Second)
	ip1 := "10.0.0.1"
	ip2 := "10.0.0.2"

	// Exhaust ip1
	for range 5 {
		rl.Allow(ip1)
	}
	allowed, _ := rl.Allow(ip1)
	if allowed {
		t.Fatalf("ip1 should be rate limited")
	}

	// ip2 should still be allowed
	allowed, _ = rl.Allow(ip2)
	if !allowed {
		t.Fatalf("ip2 should not be affected by ip1 rate limit")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	window := 50 * time.Millisecond
	rl := newRateLimiter(2, window)
	ip := "172.16.0.1"

	rl.Allow(ip)
	rl.Allow(ip)
	allowed, _ := rl.Allow(ip)
	if allowed {
		t.Fatalf("expected 3rd attempt within window to be blocked")
	}

	time.Sleep(60 * time.Millisecond)

	allowed, _ = rl.Allow(ip)
	if !allowed {
		t.Fatalf("expected attempt after window expiry to be allowed")
	}
}

func TestGetClientIP(t *testing.T) {
	// Forwarding headers from untrusted clients must not change the limiter key.
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.0.2.10:45678"
	req1.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	if ip := getClientIP(req1); ip != "192.0.2.10" {
		t.Errorf("expected peer address 192.0.2.10, got %s", ip)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.0.2.11:45678"
	req2.Header.Set("X-Real-IP", "198.51.100.1")
	if ip := getClientIP(req2); ip != "192.0.2.11" {
		t.Errorf("expected peer address 192.0.2.11, got %s", ip)
	}

	// 3. From RemoteAddr host:port
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.168.1.50:45678"
	if ip := getClientIP(req3); ip != "192.168.1.50" {
		t.Errorf("expected 192.168.1.50, got %s", ip)
	}

	// 4. From RemoteAddr bare IP
	req4 := httptest.NewRequest("GET", "/", nil)
	req4.RemoteAddr = "192.168.1.60"
	if ip := getClientIP(req4); ip != "192.168.1.60" {
		t.Errorf("expected 192.168.1.60, got %s", ip)
	}

	reqInvalid := httptest.NewRequest("GET", "/", nil)
	reqInvalid.RemoteAddr = "not-an-ip"
	reqInvalid.Header.Set("X-Forwarded-For", "203.0.113.9")
	if ip := getClientIP(reqInvalid); ip != "unknown" {
		t.Errorf("expected unknown for invalid peer address, got %s", ip)
	}
}

func TestGetClientIPUntrustedProxyIgnoresForwardedHeaders(t *testing.T) {
	resetTrustedProxyForTest()
	t.Setenv("DSWEB_TRUSTED_PROXY_CIDRS", "")
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "198.51.100.10:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 70.41.3.18")
	req.Header.Set("X-Real-IP", "203.0.113.5")
	if ip := getClientIP(req); ip != "198.51.100.10" {
		t.Errorf("untrusted proxy: expected peer address, got %s", ip)
	}
}

func TestGetClientIPHonorsTrustedProxyForwardedChain(t *testing.T) {
	resetTrustedProxyForTest()
	t.Setenv("DSWEB_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")

	if !isTrustedProxy(net.ParseIP("10.0.0.5")) {
		t.Fatalf("test setup: 10.0.0.5 should be in 10.0.0.0/8")
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.5:443"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.6")
	if ip := getClientIP(req); ip != "203.0.113.5" {
		t.Errorf("trusted proxy should walk chain to first valid IP, got %s", ip)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.5:443"
	req2.Header.Set("X-Real-IP", "203.0.113.7")
	if ip := getClientIP(req2); ip != "203.0.113.7" {
		t.Errorf("trusted proxy X-Real-IP fallback failed, got %s", ip)
	}

	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "10.0.0.5:443"
	req3.Header.Set("X-Forwarded-For", "not-an-ip")
	if ip := getClientIP(req3); ip != "10.0.0.5" {
		t.Errorf("trusted proxy without valid header should fall back to peer, got %s", ip)
	}

	trustedProxyCIDRs = nil
	trustedProxyOnce = sync.Once{}
	t.Setenv("DSWEB_TRUSTED_PROXY_CIDRS", "")
}

func TestHandleLoginRateLimit(t *testing.T) {
	server := NewServer(84)
	handler := server.Handler()

	bodyWrong, _ := json.Marshal(map[string]string{"password": "WrongPassword"})

	// Make 5 failed attempts from same IP (httptest default RemoteAddr)
	for i := range 5 {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(bodyWrong))
		req.RemoteAddr = "192.0.2.10:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rr.Code)
		}
	}

	// 6th attempt must receive HTTP 429 Too Many Requests
	req6 := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(bodyWrong))
	req6.RemoteAddr = "192.0.2.10:1234"
	rr6 := httptest.NewRecorder()
	handler.ServeHTTP(rr6, req6)

	if rr6.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt 6: expected 429 Too Many Requests, got %d (%s)", rr6.Code, rr6.Body.String())
	}

	if retryAfter := rr6.Header().Get("Retry-After"); retryAfter == "" {
		t.Errorf("expected Retry-After header on 429 response")
	}

	// Another IP should still be allowed
	reqOther := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(bodyWrong))
	reqOther.RemoteAddr = "192.0.2.20:1234"
	rrOther := httptest.NewRecorder()
	handler.ServeHTTP(rrOther, reqOther)

	if rrOther.Code != http.StatusUnauthorized {
		t.Fatalf("request from different IP expected 401, got %d", rrOther.Code)
	}
}
