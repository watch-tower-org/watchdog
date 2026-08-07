package model

import (
	"time"

	"github.com/uptrace/bun"
)

type AlertLog struct {
	bun.BaseModel `bun:"table:alert_log,alias:al"`

	ID         int64      `bun:"id,pk,autoincrement" json:"id"`
	IssueID    int64      `bun:"issue_id,notnull" json:"issue_id"`
	RuleID     int64      `bun:"rule_id,notnull" json:"rule_id"`
	SentAt     time.Time  `bun:"sent_at,notnull,default:current_timestamp" json:"sent_at"`
	Recipients []string   `bun:"recipients,type:jsonb,notnull" json:"recipients"`
	Issue      *Issue     `bun:"rel:belongs-to,join:issue_id=id" json:"issue,omitempty"`
	Rule       *AlertRule `bun:"rel:belongs-to,join:rule_id=id" json:"rule,omitempty"`
}

type ListAlertLogsRequest struct {
	Page     int   `form:"page"`
	PageSize int   `form:"page_size"`
	IssueID  int64 `form:"issue_id"`
	RuleID   int64 `form:"rule_id"`
}
