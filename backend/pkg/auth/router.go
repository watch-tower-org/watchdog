package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig, cookieSecure bool) {
	h := NewHandler(controller, cfg, cookieSecure)

	grp := r.Group("/auth")
	grp.POST("/login", h.Login)
	grp.POST("/logout", middleware.AuthMiddleware(cfg), h.Logout)
	grp.GET("/me", middleware.AuthMiddleware(cfg), h.Me)
}
