package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CORS returns middleware that validates Origin against an allowlist.
// Empty origins list disables CORS headers (same-origin only).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
				if r.Method == http.MethodOptions {
					if _, ok := allowed[origin]; ok {
						w.WriteHeader(http.StatusNoContent)
					} else {
						w.WriteHeader(http.StatusForbidden)
					}
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders sets baseline secure headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the effective client IP for rate limiting.
//
// X-Forwarded-For is only trusted when the request arrives directly from a
// configured trusted proxy; otherwise the TCP peer address is used so that
// anonymous clients cannot spoof an unlimited stream of identities.
func clientIP(r *http.Request, trustedProxies map[string]struct{}) string {
	if peer := r.RemoteAddr; peer != "" {
		if host, _, err := net.SplitHostPort(peer); err == nil {
			if _, ok := trustedProxies[host]; ok {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					parts := strings.Split(xff, ",")
					if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
						return last
					}
				}
				if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
					return strings.TrimSpace(xrip)
				}
			}
			return host
		}
	}
	return "unknown"
}

// rateLimiter is a fixed-window token bucket per IP with automatic eviction.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
}

type bucket struct {
	count int
	reset time.Time
}

func newLimiter(limit int) *rateLimiter {
	if limit <= 0 {
		limit = 120
	}
	return &rateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
	}
}

func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) > 10000 {
		// Opportunistic eviction keeps memory bounded under IP churn.
		for k, b := range l.buckets {
			if now.After(b.reset) {
				delete(l.buckets, k)
			}
		}
	}
	b, ok := l.buckets[key]
	if !ok || now.After(b.reset) {
		b = &bucket{reset: now.Add(time.Minute)}
		l.buckets[key] = b
	}
	b.count++
	return b.count <= l.limit
}

// RateLimit returns middleware applying a fixed-window per-IP rate limit.
// trustedProxies is the set of host IPs allowed to set X-Forwarded-For.
func RateLimit(requestsPerMinute int, trustedProxies []string) func(http.Handler) http.Handler {
	l := newLimiter(requestsPerMinute)
	trusted := make(map[string]struct{}, len(trustedProxies))
	for _, p := range trustedProxies {
		trusted[strings.TrimSpace(p)] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r, trusted)
			if !l.allow(ip, time.Now()) {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LoginRateLimit is a strict per-IP limiter for authentication endpoints.
func LoginRateLimit(requestsPerMinute int, trustedProxies []string) func(http.Handler) http.Handler {
	return RateLimit(requestsPerMinute, trustedProxies)
}
