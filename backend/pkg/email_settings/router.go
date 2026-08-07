package email_settings

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/settings/email")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.Get)
	grp.PUT("", h.Update)
	grp.GET("/test", h.TestEmail)
	grp.POST("/test", h.TestEmail)
}
