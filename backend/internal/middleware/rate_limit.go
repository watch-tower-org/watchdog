package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

func RateLimit(cfg *config.RateLimitConfig) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: cfg.Window,
		Limit:  int64(cfg.Requests),
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return func(c *gin.Context) {
		key := c.ClientIP()
		ctx := c.Request.Context()
		limitCtx, err := instance.Get(ctx, key)
		if err != nil {
			c.Next()
			return
		}

		if limitCtx.Reached {
			res.TooManyRequests(c, "Too many requests. Please try again later.")
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(limitCtx.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(limitCtx.Reset, 10))

		c.Next()
	}
}
