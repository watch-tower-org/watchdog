package model

import (
	"time"

	"github.com/uptrace/bun"
)

type IssueStatus string

const (
	IssueStatusOpen     IssueStatus = "open"
	IssueStatusResolved IssueStatus = "resolved"
	IssueStatusMuted    IssueStatus = "muted"
)

type Issue struct {
	bun.BaseModel `bun:"table:issues,alias:i"`

	ID          int64       `bun:"id,pk,autoincrement" json:"id"`
	Fingerprint string      `bun:"fingerprint,unique,notnull" json:"fingerprint"`
	Title       string      `bun:"title,notnull" json:"title"`
	Project     string      `bun:"project,notnull" json:"project"`
	Tag         string      `bun:"tag" json:"tag"`
	Status      IssueStatus `bun:"status,notnull,default:'open'" json:"status"`
	FirstSeen   time.Time   `bun:"first_seen,notnull,default:current_timestamp" json:"first_seen"`
	LastSeen    time.Time   `bun:"last_seen,notnull,default:current_timestamp" json:"last_seen"`
	Count       int64       `bun:"count,notnull,default:0" json:"count"`
	CreatedAt   time.Time   `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time   `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type ListIssuesRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Project  string `form:"project"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

type UpdateIssueRequest struct {
	Status *IssueStatus `json:"status" validate:"omitempty,oneof=open resolved muted"`
}
