package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

var (
	ErrInvalidCredentials = errors.New("Invalid username or password")
)

type Controller struct {
	db  *bun.DB
	cfg *config.JWTConfig
}

func NewController(db *bun.DB, cfg *config.JWTConfig) *Controller {
	return &Controller{db: db, cfg: cfg}
}

func (c *Controller) HashPassword(pwd string) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Error().Msgf("HashPassword: bcrypt failed: %v", err)
		return "", errors.New("An unexpected error occurred. Please try again.")
	}
	return string(hashedPwd), nil
}

func (c *Controller) loadSettings(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	err := c.db.NewSelect().
		Model(&s).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("auth: failed to load settings: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return &s, nil
}

func (c *Controller) generateToken(username string, key string, duration time.Duration) (string, error) {
	claims := &model.Claims{
		Id:       1,
		Username: username,
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    "watchtower",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", 1),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		logger.Error().Msgf("failed to sign JWT: %v", err)
		return "", errors.New("An unexpected error occurred. Please try again.")
	}
	return tokenString, nil
}

func (c *Controller) checkPassword(hashedPwd string, requestPwd string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(requestPwd)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (c *Controller) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	if s.AdminUsername == "" || s.AdminPassword == "" {
		return nil, errors.New("Admin account is not configured.")
	}

	if s.AdminUsername != req.Username {
		return nil, ErrInvalidCredentials
	}

	if err := c.checkPassword(s.AdminPassword, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := c.generateToken(s.AdminUsername, c.cfg.SecretKey, c.cfg.ExpirationDuration)
	if err != nil {
		return nil, err
	}
	refreshToken, err := c.generateToken(s.AdminUsername, c.cfg.RefreshSecretKey, c.cfg.RefreshExpirationDuration)
	if err != nil {
		return nil, err
	}

	resp := &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     s.AdminUsername,
	}

	return resp, nil
}

func (c *Controller) RefreshToken(ctx context.Context, username string) (*model.RefreshTokenResponse, error) {
	accessToken, err := c.generateToken(username, c.cfg.SecretKey, c.cfg.ExpirationDuration)
	if err != nil {
		return nil, err
	}

	return &model.RefreshTokenResponse{
		AccessToken: accessToken,
	}, nil
}
