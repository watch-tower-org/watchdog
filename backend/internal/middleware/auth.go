package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

func ValidateJWT(token, secret string) (*model.Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := parsedToken.Claims.(*model.Claims); ok && parsedToken.Valid {
		if claims.Issuer != "" && claims.Issuer != "watchtower" {
			return nil, errors.New("invalid token issuer")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func AuthMiddleware(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg == nil {
			res.InternalServerError(c, "auth configuration not initialized")
			c.Abort()
			return
		}

		// Prefer the Authorization header (API clients/curl), fall back to the
		// httpOnly access-token cookie used by the SPA.
		var split string
		token := c.GetHeader("Authorization")
		switch {
		case strings.HasPrefix(token, "Bearer "):
			split = strings.TrimSpace(token[len("Bearer "):])
		case token != "":
			res.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		default:
			split, _ = c.Cookie(AccessTokenCookie)
		}

		if split == "" {
			res.Unauthorized(c, "Authorization is required")
			c.Abort()
			return
		}

		claims, err := ValidateJWT(split, cfg.SecretKey)
		if err != nil {
			res.Unauthorized(c, "Invalid authorization token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.Id)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}
