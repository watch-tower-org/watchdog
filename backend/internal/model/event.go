package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Event struct {
	bun.BaseModel `bun:"table:events,alias:e"`

	ID         int64          `bun:"id,pk,autoincrement" json:"id"`
	IssueID    int64          `bun:"issue_id,notnull" json:"issue_id"`
	Timestamp  time.Time      `bun:"timestamp,notnull,default:current_timestamp" json:"timestamp"`
	Message    string         `bun:"message,type:text" json:"message"`
	StackTrace string         `bun:"stack_trace,type:text" json:"stack_trace"`
	Context    map[string]any `bun:"context,type:jsonb" json:"context"`
	Project    string         `bun:"project,notnull" json:"project"`
	Tag        string         `bun:"tag" json:"tag"`
	Issue      *Issue         `bun:"rel:belongs-to,join:issue_id=id" json:"issue,omitempty"`
	CreatedAt  time.Time      `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

type IngestEventRequest struct {
	Message    string         `json:"message" validate:"required"`
	ErrorType  string         `json:"error_type"`
	StackTrace string         `json:"stack_trace"`
	Project    string         `json:"project"`
	Tag        string         `json:"tag"`
	Context    map[string]any `json:"context"`
	Timestamp  *time.Time     `json:"timestamp"`
}

type IngestResult struct {
	IssueID     int64  `json:"issue_id"`
	EventID     int64  `json:"event_id"`
	IsNewIssue  bool   `json:"is_new_issue"`
	Fingerprint string `json:"fingerprint"`
}

type ListEventsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	IssueID  int64  `form:"issue_id"`
	Project  string `form:"project"`
}
