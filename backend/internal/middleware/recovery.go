package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

// RecoverFn is called with the recovered panic value and request context. It
// may be nil (no reporting).
type RecoverFn func(v any, ctx map[string]any)

// Recovery recovers panics from downstream handlers, logs them, reports them
// via the optional self-reporting callback, and responds 500 so the server
// stays alive.
func Recovery(report RecoverFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if report != nil {
					report(err, map[string]any{
						"url":       c.Request.URL.Path,
						"method":    c.Request.Method,
						"client_ip": c.ClientIP(),
					})
				}
				logger.Error().
					Interface("panic", err).
					Bytes("stack", debug.Stack()).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")
				res.InternalServerError(c, "An unexpected error occurred. Please try again.")
				c.Abort()
			}
		}()
		c.Next()
	}
}
