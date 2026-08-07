package model

import (
	"time"

	"github.com/uptrace/bun"
)

type EmailSettings struct {
	bun.BaseModel `bun:"table:email_settings,alias:es"`

	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	SMTPHost      string    `bun:"smtp_host" json:"smtp_host"`
	SMTPPort      int       `bun:"smtp_port" json:"smtp_port"`
	SMTPUsername  string    `bun:"smtp_username" json:"smtp_username"`
	SMTPPassword  string    `bun:"smtp_password" json:"-"`
	SMTPFromEmail string    `bun:"smtp_from_email" json:"smtp_from_email"`
	SMTPFromName  string    `bun:"smtp_from_name" json:"smtp_from_name"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type UpdateEmailSettingsRequest struct {
	SMTPHost      *string `json:"smtp_host"`
	SMTPPort      *int    `json:"smtp_port"`
	SMTPUsername  *string `json:"smtp_username"`
	SMTPPassword  *string `json:"smtp_password"`
	SMTPFromEmail *string `json:"smtp_from_email"`
	SMTPFromName  *string `json:"smtp_from_name"`
}

type TestEmailRequest struct {
	To            string  `json:"to" validate:"required,email"`
	SMTPHost      *string `json:"smtp_host"`
	SMTPPort      *int    `json:"smtp_port"`
	SMTPUsername  *string `json:"smtp_username"`
	SMTPPassword  *string `json:"smtp_password"`
	SMTPFromEmail *string `json:"smtp_from_email"`
	SMTPFromName  *string `json:"smtp_from_name"`
}
