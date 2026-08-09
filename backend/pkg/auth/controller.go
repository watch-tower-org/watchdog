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

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

var (
	ErrInvalidCredentials = errors.New("Invalid username or password")
	ErrInvalidRefreshToken = errors.New("Invalid refresh token")
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

func (c *Controller) generateToken(username string, adminID int64, jti string, key string, duration time.Duration) (string, error) {
	claims := &model.Claims{
		Id:       adminID,
		Username: username,
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "watchtower",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", adminID),
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

	// Opportunistically drop this admin's expired sessions so rotation and
	// re-login don't accumulate rows forever.
	c.purgeExpiredSessions(ctx, a.ID)

	refreshJTI := uuid.New().String()
	refreshExpiresAt := time.Now().Add(c.cfg.RefreshExpirationDuration)
	session := &model.AuthSession{
		JTI:       refreshJTI,
		AdminID:   a.ID,
		ExpiresAt: refreshExpiresAt,
	}
	if _, err := c.db.NewInsert().Model(session).Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("auth: failed to persist refresh session for %q: %v", req.Username, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	accessToken, err := c.generateToken(a.Username, a.ID, uuid.New().String(), c.cfg.SecretKey, c.cfg.ExpirationDuration)
	if err != nil {
		return nil, err
	}
	refreshToken, err := c.generateToken(a.Username, a.ID, refreshJTI, c.cfg.RefreshSecretKey, c.cfg.RefreshExpirationDuration)
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

// createSession inserts a new session row and returns its JTI.
func (c *Controller) createSession(ctx context.Context, adminID int64, expiresAt time.Time) (string, error) {
	jti := uuid.New().String()
	session := &model.AuthSession{
		JTI:       jti,
		AdminID:   adminID,
		ExpiresAt: expiresAt,
	}
	if _, err := c.db.NewInsert().Model(session).Exec(ctx); err != nil {
		return "", err
	}
	return jti, nil
}

// rotateSession revokes the given session row and creates its replacement,
// returning the new JTI.
func (c *Controller) rotateSession(ctx context.Context, session *model.AuthSession, expiresAt time.Time) (string, error) {
	newJTI, err := c.createSession(ctx, session.AdminID, expiresAt)
	if err != nil {
		return "", err
	}

	if _, err := c.db.NewUpdate().
		Model((*model.AuthSession)(nil)).
		Set("revoked_at = ?", time.Now()).
		Set("replaced_by_jti = ?", newJTI).
		Where("id = ?", session.ID).
		Where("revoked_at IS NULL").
		Exec(ctx); err != nil {
		return "", err
	}

	return newJTI, nil
}

// purgeExpiredSessions deletes expired session rows for an admin so the table
// doesn't grow unbounded. Best-effort; errors are logged and ignored.
func (c *Controller) purgeExpiredSessions(ctx context.Context, adminID int64) {
	if _, err := c.db.NewDelete().
		Model((*model.AuthSession)(nil)).
		Where("admin_id = ?", adminID).
		Where("expires_at < ?", time.Now()).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Warn().Msgf("auth: failed to purge expired sessions for admin %d: %v", adminID, err)
	}
}

// RevokeSession invalidates the refresh session backing a refresh token. Used
// by logout so a stolen refresh cookie stops working server-side, not just in
// the browser.
func (c *Controller) RevokeSession(ctx context.Context, refreshJWT string) error {
	claims, err := middleware.ValidateJWT(refreshJWT, c.cfg.RefreshSecretKey)
	if err != nil {
		return err
	}
	if claims.ID == "" {
		return ErrInvalidRefreshToken
	}

	_, err = c.db.NewUpdate().
		Model((*model.AuthSession)(nil)).
		Set("revoked_at = ?", time.Now()).
		Where("jti = ?", claims.ID).
		Where("revoked_at IS NULL").
		Exec(ctx)
	return err
}

// EnsureAdmin creates the admin account from env-provided credentials on
// first boot. An existing account is left untouched unless reset is true
// (opt-in via ADMIN_RESET_PASSWORD), so a default or stale ADMIN_PASSWORD in
// the environment can never silently clobber the admin password.
func (c *Controller) EnsureAdmin(username, password string, reset bool) error {
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
		if !reset {
			return nil
		}
		existing.Password = hashedPwd
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

// RefreshToken validates a refresh token against its server-side session,
// rejects revoked/expired sessions and inactive admins, then rotates the
// session (revoking the presented token) and issues fresh access + refresh
// tokens.
func (c *Controller) RefreshToken(ctx context.Context, refreshJWT string) (*model.RefreshTokenResponse, error) {
	claims, err := middleware.ValidateJWT(refreshJWT, c.cfg.RefreshSecretKey)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	if claims.ID == "" {
		return nil, ErrInvalidRefreshToken
	}

	var session model.AuthSession
	err = c.db.NewSelect().
		Model(&session).
		Where("jti = ?", claims.ID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		logger.Ctx(ctx).Error().Msgf("auth: failed to load session %q: %v", claims.ID, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	now := time.Now()
	if session.RevokedAt != nil || now.After(session.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	// Re-load the admin so a deactivated account can't keep refreshing.
	var admin model.Admin
	err = c.db.NewSelect().
		Model(&admin).
		Where("id = ?", session.AdminID).
		Where("is_active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	refreshExpiresAt := now.Add(c.cfg.RefreshExpirationDuration)
	newJTI, err := c.rotateSession(ctx, &session, refreshExpiresAt)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("auth: failed to rotate session %q: %v", claims.ID, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	accessToken, err := c.generateToken(admin.Username, admin.ID, uuid.New().String(), c.cfg.SecretKey, c.cfg.ExpirationDuration)
	if err != nil {
		return nil, err
	}
	newRefreshToken, err := c.generateToken(admin.Username, admin.ID, newJTI, c.cfg.RefreshSecretKey, c.cfg.RefreshExpirationDuration)
	if err != nil {
		return nil, err
	}

	return &model.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Username:     admin.Username,
	}, nil
}
