package settings

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/settings")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.Get)
	grp.PUT("", h.Update)
}
