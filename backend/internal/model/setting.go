package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Settings struct {
	bun.BaseModel `bun:"table:settings,alias:s"`

	ID              int64     `bun:"id,pk,autoincrement" json:"id"`
	SetupComplete   bool      `bun:"setup_complete,notnull,default:false" json:"setup_complete"`
	AdminUsername   string    `bun:"admin_username,notnull" json:"-"`
	AdminPassword   string    `bun:"admin_password,notnull" json:"-"` // bcrypt hash
	APIKeyHash      string    `bun:"api_key_hash" json:"-"`           // SHA-256 of global API key
	APIKeyPlaintext string    `bun:"-" json:"api_key,omitempty"`      // shown once at setup, never persisted
	SMTPHost        string    `bun:"smtp_host" json:"smtp_host"`
	SMTPPort        int       `bun:"smtp_port" json:"smtp_port"`
	SMTPUsername    string    `bun:"smtp_username" json:"smtp_username"`
	SMTPPassword    string    `bun:"smtp_password" json:"-"`
	SMTPFromEmail   string    `bun:"smtp_from_email" json:"smtp_from_email"`
	SMTPFromName    string    `bun:"smtp_from_name" json:"smtp_from_name"`
	ThrottleWindow  int       `bun:"throttle_window,notnull,default:60" json:"throttle_window"`
	CreatedAt       time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type UpdateEmailSettingsRequest struct {
	SMTPHost      *string `json:"smtp_host"`
	SMTPPort      *int    `json:"smtp_port"`
	SMTPUsername  *string `json:"smtp_username"`
	SMTPPassword  *string `json:"smtp_password"`
	SMTPFromEmail *string `json:"smtp_from_email"`
	SMTPFromName  *string `json:"smtp_from_name"`
}

type UpdateThrottleRequest struct {
	ThrottleWindow *int `json:"throttle_window" validate:"required,min=1"`
}
