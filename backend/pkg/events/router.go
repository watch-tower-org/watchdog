package events

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/events")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.List)
	grp.GET("/:id", h.Get)
}
