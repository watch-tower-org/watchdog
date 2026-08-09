package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
)

// newRateLimitRouter builds a router with only the rate limiter mounted, using
// the given trusted proxies.
func newRateLimitRouter(trusted []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies(trusted); err != nil {
		panic(err)
	}
	cfg := &config.RateLimitConfig{Requests: 2, Window: time.Minute}
	r.Use(RateLimit(cfg))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

// TestRateLimitIgnoresSpoofedForwardedFor is the security regression test: with
// no trusted proxies configured, X-Forwarded-For must NOT be honored, so an
// attacker can't reset their rate-limit bucket by spoofing the header. Requests
// from the same real remote IP share one bucket regardless of the spoofed
// header value.
func TestRateLimitIgnoresSpoofedForwardedFor(t *testing.T) {
	r := newRateLimitRouter(nil)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "198.51.100.10:12345"
		// Each attempt claims a fresh attacker-chosen client IP.
		req.Header.Set("X-Forwarded-For", "203.0.113.42")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, w.Code)
		}
	}

	// Third request from the same real IP must be rate limited even though the
	// spoofed header differs.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "198.51.100.10:56789"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: want 429, got %d", w.Code)
	}
}

// TestRateLimitHonorsForwardedFromTrustedProxy confirms that when a proxy CIDR
// is explicitly trusted, X-Forwarded-For is honored and keys the bucket.
func TestRateLimitHonorsForwardedFromTrustedProxy(t *testing.T) {
	r := newRateLimitRouter([]string{"192.168.0.0/16"})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "192.168.1.10:8080" // a trusted proxy
		req.Header.Set("X-Forwarded-For", "203.0.113.10")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		want := http.StatusOK
		if i >= 2 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("request %d: want %d, got %d", i, want, w.Code)
		}
	}
}
