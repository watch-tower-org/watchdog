package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Settings struct {
	bun.BaseModel `bun:"table:settings,alias:s"`

	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	SetupComplete bool      `bun:"setup_complete,notnull,default:false" json:"setup_complete"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type UpdateSettingsRequest struct {
	SetupComplete *bool `json:"setup_complete"`
}
