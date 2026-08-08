package model

import (
	"time"

	"github.com/uptrace/bun"
)

type ApiKey struct {
	bun.BaseModel `bun:"table:api_keys,alias:ak"`

	ID         int64      `bun:"id,pk,autoincrement" json:"id"`
	Name       string     `bun:"name,notnull" json:"name"`
	KeyHash    string     `bun:"key_hash,unique,notnull" json:"-"`
	Masked     string     `bun:"masked,notnull,default:''" json:"masked"`
	Project    string     `bun:"project,notnull,default:''" json:"project"`
	IsActive   bool       `bun:"is_active,notnull,default:true" json:"is_active"`
	LastUsedAt *time.Time `bun:"last_used_at,nullzero" json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type CreateApiKeyRequest struct {
	Name    string `json:"name" validate:"required"`
	Project string `json:"project"`
}

type UpdateApiKeyRequest struct {
	Name    *string `json:"name"`
	Project *string `json:"project"`
}

type CreateApiKeyResponse struct {
	ApiKey ApiKey `json:"api_key"`
	Key    string `json:"key"` // plaintext, shown only once
	KeyID  int64  `json:"key_id"`
}

type ListApiKeysRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}
