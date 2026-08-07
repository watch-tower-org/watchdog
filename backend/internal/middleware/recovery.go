package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/res"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
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
