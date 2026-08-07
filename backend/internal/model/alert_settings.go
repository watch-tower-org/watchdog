package model

import (
	"time"

	"github.com/uptrace/bun"
)

type AlertSettings struct {
	bun.BaseModel `bun:"table:alert_settings,alias:as"`

	ID             int64     `bun:"id,pk,autoincrement" json:"id"`
	ThrottleWindow int       `bun:"throttle_window,notnull,default:60" json:"throttle_window"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type UpdateAlertSettingsRequest struct {
	ThrottleWindow *int `json:"throttle_window" validate:"required,min=1"`
}
