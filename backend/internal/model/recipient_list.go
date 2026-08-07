package model

import (
	"time"

	"github.com/uptrace/bun"
)

type RecipientList struct {
	bun.BaseModel `bun:"table:recipient_lists,alias:rl"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	Name      string    `bun:"name,notnull" json:"name"`
	Emails    []string  `bun:"emails,type:jsonb,notnull" json:"emails"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type CreateRecipientListRequest struct {
	Name   string   `json:"name" validate:"required"`
	Emails []string `json:"emails" validate:"required,min=1,dive,email"`
}

type UpdateRecipientListRequest struct {
	Name   *string   `json:"name"`
	Emails *[]string `json:"emails" validate:"omitempty,min=1,dive,email"`
}

type ListRecipientListsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
}
