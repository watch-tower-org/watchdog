package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bun"
)

type Claims struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// AuthSession tracks a server-side refresh-token session so tokens can be
// revoked, rotated, and checked against the admin's current state. The JTI of
// the refresh JWT is stored here; rotation marks the old row revoked and
// records the replacement.
type AuthSession struct {
	bun.BaseModel `bun:"table:auth_sessions,alias:ses"`

	ID         int64      `bun:"id,pk,autoincrement" json:"id"`
	JTI        string     `bun:"jti,unique,notnull" json:"jti"`
	AdminID    int64      `bun:"admin_id,notnull" json:"admin_id"`
	ExpiresAt  time.Time  `bun:"expires_at,notnull" json:"expires_at"`
	RevokedAt  *time.Time `bun:"revoked_at" json:"revoked_at"`
	ReplacedBy *string    `bun:"replaced_by_jti" json:"replaced_by_jti"`
	CreatedAt  time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// LoginResponse carries the session tokens back to the handler, which writes
// them to httpOnly cookies. The tokens are never serialized into the JSON
// response body.
type LoginResponse struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	Username     string `json:"username"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	Username     string `json:"username"`
}

type PageInfo struct {
	CurrentPage     int  `json:"current_page"`
	Limit           int  `json:"limit"`
	Total           int  `json:"total"`
	TotalPages      int  `json:"total_pages"`
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
}
