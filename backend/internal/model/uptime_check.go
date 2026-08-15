package model

import (
	"time"

	"github.com/uptrace/bun"
)

type UptimeCheck struct {
	bun.BaseModel `bun:"table:uptime_checks,alias:uc"`

	ID             int64         `bun:"id,pk,autoincrement" json:"id"`
	ServiceID      int64         `bun:"service_id,notnull" json:"service_id"`
	CheckedAt      time.Time     `bun:"checked_at,notnull" json:"checked_at"`
	Status         ServiceStatus `bun:"status,notnull" json:"status"`
	StatusCode     *int          `bun:"status_code" json:"status_code"`
	ResponseTimeMs *int          `bun:"response_time_ms" json:"response_time_ms"`
	Error          *string       `bun:"error" json:"error"`
}
