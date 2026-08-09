package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
	h := NewHandler(controller, cfg)

	grp := r.Group("/auth")
	grp.POST("/login", h.Login)
	grp.POST("/refresh-token", h.RefreshToken)
	grp.POST("/logout", middleware.AuthMiddleware(cfg), h.Logout)
}
