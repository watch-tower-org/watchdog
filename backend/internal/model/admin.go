package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Admin struct {
	bun.BaseModel `bun:"table:admins,alias:ad"`

	ID          int64     `bun:"id,pk,autoincrement" json:"id"`
	Username    string    `bun:"username,unique,notnull" json:"username"`
	Password    string    `bun:"password,notnull" json:"-"`
	IsProtected bool      `bun:"is_protected,notnull,default:false" json:"is_protected"`
	IsActive    bool      `bun:"is_active,notnull,default:true" json:"is_active"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}
