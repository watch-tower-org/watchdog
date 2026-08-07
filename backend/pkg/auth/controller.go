package auth

import (
	"context"
	"database/sql"
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

func (c *Controller) loadAdmin(ctx context.Context, username string) (*model.Admin, error) {
	var a model.Admin
	err := c.db.NewSelect().
		Model(&a).
		Where("username = ?", username).
		Where("is_active = ?", true).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("auth: failed to load admin %q: %v", username, err)
		return nil, errors.New("Invalid username or password")
	}

	return &a, nil
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
	a, err := c.loadAdmin(ctx, req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := c.checkPassword(a.Password, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := c.generateToken(a.Username, c.cfg.SecretKey, c.cfg.ExpirationDuration)
	if err != nil {
		return nil, err
	}
	refreshToken, err := c.generateToken(a.Username, c.cfg.RefreshSecretKey, c.cfg.RefreshExpirationDuration)
	if err != nil {
		return nil, err
	}

	resp := &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     a.Username,
	}

	return resp, nil
}

// EnsureAdmin creates or updates the admin account from env-provided
// credentials (upsert by username) so password changes propagate on restart.
func (c *Controller) EnsureAdmin(username, password string) error {
	ctx := context.Background()

	hashedPwd, err := c.HashPassword(password)
	if err != nil {
		return err
	}

	var existing model.Admin
	err = c.db.NewSelect().
		Model(&existing).
		Where("username = ?", username).
		Scan(ctx)

	if err == nil {
		existing.Password = hashedPwd
		existing.IsActive = true
		existing.UpdatedAt = time.Now()
		_, err = c.db.NewUpdate().
			Model(&existing).
			Where("id = ?", existing.ID).
			Exec(ctx)
		if err != nil {
			logger.Ctx(ctx).Error().Msgf("EnsureAdmin: update failed for %q: %v", username, err)
			return errors.New("An unexpected error occurred. Please try again.")
		}
		return nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		logger.Ctx(ctx).Error().Msgf("EnsureAdmin: select failed for %q: %v", username, err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	admin := &model.Admin{
		Username:    username,
		Password:    hashedPwd,
		IsProtected: true,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = c.db.NewInsert().Model(admin).Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("EnsureAdmin: insert failed for %q: %v", username, err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	return nil
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
