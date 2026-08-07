package api_keys

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/api-keys")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.List)
	grp.POST("", h.Create)
	grp.PUT("/:id/revoke", h.Revoke)
}
