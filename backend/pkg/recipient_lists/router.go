package recipient_lists

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller)

	grp := r.Group("/recipient-lists")
	grp.Use(middleware.AuthMiddleware(cfg))

	grp.GET("", h.List)
	grp.GET("/:id", h.Get)
	grp.POST("", h.Create)
	grp.PUT("/:id", h.Update)
	grp.DELETE("/:id", h.Delete)
}
