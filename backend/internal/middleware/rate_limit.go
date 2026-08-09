package middleware

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

// newLimiter builds an in-memory limiter from the rate limit config.
func newLimiter(cfg *config.RateLimitConfig) *limiter.Limiter {
	rate := limiter.Rate{
		Period: cfg.Window,
		Limit:  int64(cfg.Requests),
	}
	return limiter.New(memory.NewStore(), rate)
}

// consume applies the limiter for key; it writes the rate-limit headers and, on
// reaching the limit, responds 429 and aborts the request. It returns true when
// the request was aborted.
func consume(c *gin.Context, instance *limiter.Limiter, key string, cfg *config.RateLimitConfig) bool {
	limitCtx, err := instance.Get(c.Request.Context(), key)
	if err != nil {
		// Fail open on limiter store errors so an unavailable limiter never
		// blocks legitimate traffic.
		return false
	}

	if limitCtx.Reached {
		res.TooManyRequests(c, "Too many requests. Please try again later.")
		c.Abort()
		return true
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
	c.Header("X-RateLimit-Remaining", strconv.FormatInt(limitCtx.Remaining, 10))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(limitCtx.Reset, 10))
	return false
}

// RateLimit is a per-client-IP limiter applied to the whole API. The key is the
// real client IP, which only trusts X-Forwarded-For from proxies configured via
// TRUSTED_PROXIES (see router.AppRouter).
func RateLimit(cfg *config.RateLimitConfig) gin.HandlerFunc {
	instance := newLimiter(cfg)
	return func(c *gin.Context) {
		if consume(c, instance, c.ClientIP(), cfg) {
			return
		}
		c.Next()
	}
}

// ApiKeyRateLimit is an additional per-API-key limiter for the ingestion route.
// It must run after ApiKeyAuth so the authenticated key ID is available in the
// context; a noisy SDK key can then never consume another key's or client's
// budget.
func ApiKeyRateLimit(cfg *config.RateLimitConfig) gin.HandlerFunc {
	instance := newLimiter(cfg)
	return func(c *gin.Context) {
		keyID, ok := c.Get(CtxApiKeyID)
		if !ok {
			c.Next()
			return
		}
		id, ok := keyID.(int64)
		if !ok {
			c.Next()
			return
		}
		if consume(c, instance, fmt.Sprintf("apikey:%d", id), cfg) {
			return
		}
		c.Next()
	}
}
