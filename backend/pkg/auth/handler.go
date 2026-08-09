package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
)

type Handler struct {
	controller *Controller
	cfg        *config.JWTConfig
}

func NewHandler(controller *Controller, cfg *config.JWTConfig) *Handler {
	return &Handler{controller: controller, cfg: cfg}
}

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, "Username and password are required")
		return
	}

	req.Username = validator.SanitizeString(req.Username)

	u, err := h.controller.Login(c.Request.Context(), &req)
	if err != nil {
		res.Unauthorized(c, "Invalid username or password")
		return
	}

	res.Ok(c, "User logged in successfully", u)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, "Refresh token is required")
		return
	}

	claims, err := middleware.ValidateJWT(req.RefreshToken, h.cfg.RefreshSecretKey)
	if err != nil {
		res.Unauthorized(c, "Invalid refresh token")
		return
	}

	u, err := h.controller.RefreshToken(c.Request.Context(), claims.Username)
	if err != nil {
		res.Unauthorized(c, "Invalid refresh token")
		return
	}

	res.Ok(c, "Token refreshed successfully", u)
}

func (h *Handler) Logout(c *gin.Context) {
	if _, exists := c.Get("username"); !exists {
		res.Unauthorized(c, "Unauthorized")
		return
	}

	res.Ok(c, "User logged out successfully", nil)
}
