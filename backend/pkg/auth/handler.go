package auth

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
)

func intSeconds(d time.Duration) int {
	return int(d.Seconds())
}

type Handler struct {
	controller   *Controller
	cfg          *config.JWTConfig
	cookieSecure bool
}

func NewHandler(controller *Controller, cfg *config.JWTConfig, cookieSecure bool) *Handler {
	return &Handler{controller: controller, cfg: cfg, cookieSecure: cookieSecure}
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

	middleware.SetAuthCookies(
		c,
		u.AccessToken,
		u.RefreshToken,
		intSeconds(h.cfg.ExpirationDuration),
		intSeconds(h.cfg.RefreshExpirationDuration),
		h.cookieSecure,
	)

	res.Ok(c, "User logged in successfully", u)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	// The refresh token comes from the httpOnly cookie, not the request body.
	refreshToken, _ := c.Cookie(middleware.RefreshTokenCookie)
	if refreshToken == "" {
		res.Unauthorized(c, "Refresh token is required")
		return
	}

	claims, err := middleware.ValidateJWT(refreshToken, h.cfg.RefreshSecretKey)
	if err != nil {
		res.Unauthorized(c, "Invalid refresh token")
		return
	}

	u, err := h.controller.RefreshToken(c.Request.Context(), claims.Username)
	if err != nil {
		res.Unauthorized(c, "Invalid refresh token")
		return
	}

	middleware.RefreshAccessCookie(c, u.AccessToken, intSeconds(h.cfg.ExpirationDuration), h.cookieSecure)

	res.Ok(c, "Token refreshed successfully", u)
}

func (h *Handler) Logout(c *gin.Context) {
	if _, exists := c.Get("username"); !exists {
		res.Unauthorized(c, "Unauthorized")
		return
	}

	middleware.ClearAuthCookies(c)

	res.Ok(c, "User logged out successfully", nil)
}

func (h *Handler) Me(c *gin.Context) {
	username, _ := c.Get("username")
	if username == nil {
		res.Unauthorized(c, "Unauthorized")
		return
	}

	res.Ok(c, "User retrieved successfully", map[string]string{
		"username": username.(string),
	})
}
