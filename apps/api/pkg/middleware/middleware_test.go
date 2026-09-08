package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPIgnoresSpoofedXFF(t *testing.T) {
	trusted := map[string]struct{}{"172.30.0.20": {}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:51234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	if got := clientIP(req, trusted); got != "203.0.113.7" {
		t.Errorf("untrusted peer must not use XFF: got %q", got)
	}
}

func TestClientIPTrustsXFFFromTrustedProxy(t *testing.T) {
	trusted := map[string]struct{}{"172.30.0.20": {}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.30.0.20:443"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	if got := clientIP(req, trusted); got != "203.0.113.9" {
		t.Errorf("trusted proxy XFF not honored: got %q", got)
	}
}

func TestClientIPTakesLastXFFHop(t *testing.T) {
	trusted := map[string]struct{}{"172.30.0.20": {}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.30.0.20:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.9")

	if got := clientIP(req, trusted); got != "203.0.113.9" {
		t.Errorf("expected rightmost real client, got %q", got)
	}
}

func TestRateLimiterAllowsWithinWindow(t *testing.T) {
	l := newLimiter(3)
	now := time.Now()
	for i := 1; i <= 3; i++ {
		if !l.allow("1.2.3.4", now) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if l.allow("1.2.3.4", now) {
		t.Fatal("4th request should be denied")
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	l := newLimiter(2)
	now := time.Now()
	l.allow("1.2.3.4", now)
	l.allow("1.2.3.4", now)
	if l.allow("1.2.3.4", now) {
		t.Fatal("should be denied inside window")
	}
	if !l.allow("1.2.3.4", now.Add(61*time.Second)) {
		t.Fatal("should reset after window elapsed")
	}
}

func TestRateLimiterKeysArePerIP(t *testing.T) {
	l := newLimiter(1)
	now := time.Now()
	if !l.allow("1.1.1.1", now) {
		t.Fatal("first IP allowed")
	}
	if !l.allow("2.2.2.2", now) {
		t.Fatal("second IP must have own bucket")
	}
}

func TestRateLimitSkipsHealthEndpoints(t *testing.T) {
	trusted := []string{}
	handler := RateLimit(0, trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/ready should bypass rate limit, got %d", rec.Code)
	}
}

func TestRateLimitBlockedReturns429(t *testing.T) {
	handler := RateLimit(1, []string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
		req.RemoteAddr = "203.0.113.99:1"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first request should pass, got %d", rec.Code)
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second request should be 429, got %d", rec.Code)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Permissions-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing header %s", h)
		}
	}
}
