package settings

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/settings")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.GetSettings)
	grp.GET("/email", h.GetEmailSettings)
	grp.PUT("/email", h.UpdateEmailSettings)
	grp.GET("/email/test", h.TestEmailSettings)
	grp.PUT("/throttle", h.UpdateThrottle)
}
