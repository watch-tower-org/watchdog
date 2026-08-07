package model

import (
	"time"

	"github.com/uptrace/bun"
)

type AlertTriggerType string

const (
	AlertTriggerNewIssue   AlertTriggerType = "new_issue"
	AlertTriggerSpike      AlertTriggerType = "spike"
	AlertTriggerRegression AlertTriggerType = "regression"
)

type AlertRule struct {
	bun.BaseModel `bun:"table:alert_rules,alias:ar"`

	ID              int64            `bun:"id,pk,autoincrement" json:"id"`
	Name            string           `bun:"name,notnull" json:"name"`
	TriggerType     AlertTriggerType `bun:"trigger_type,notnull" json:"trigger_type"`
	Project         string           `bun:"project" json:"project"`
	Tag             string           `bun:"tag" json:"tag"`
	Threshold       int              `bun:"threshold" json:"threshold"`
	WindowMinutes   int              `bun:"window_minutes" json:"window_minutes"`
	ThrottleWindow  int              `bun:"throttle_window,notnull,default:60" json:"throttle_window"`
	RecipientListID int64            `bun:"recipient_list_id,notnull" json:"recipient_list_id"`
	RecipientList   *RecipientList   `bun:"rel:belongs-to,join:recipient_list_id=id" json:"recipient_list,omitempty"`
	IsActive        bool             `bun:"is_active,notnull,default:true" json:"is_active"`
	CreatedAt       time.Time        `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time        `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type CreateAlertRuleRequest struct {
	Name            string           `json:"name" validate:"required"`
	TriggerType     AlertTriggerType `json:"trigger_type" validate:"required,oneof=new_issue spike regression"`
	Project         string           `json:"project"`
	Tag             string           `json:"tag"`
	Threshold       int              `json:"threshold"`
	WindowMinutes   int              `json:"window_minutes"`
	ThrottleWindow  int              `json:"throttle_window" validate:"omitempty,min=1"`
	RecipientListID int64            `json:"recipient_list_id" validate:"required"`
	IsActive        *bool            `json:"is_active"`
}

type UpdateAlertRuleRequest struct {
	Name            *string           `json:"name"`
	TriggerType     *AlertTriggerType `json:"trigger_type"`
	Project         *string           `json:"project"`
	Tag             *string           `json:"tag"`
	Threshold       *int              `json:"threshold"`
	WindowMinutes   *int              `json:"window_minutes"`
	ThrottleWindow  *int              `json:"throttle_window"`
	RecipientListID *int64            `json:"recipient_list_id"`
	IsActive        *bool             `json:"is_active"`
}

type ListAlertRulesRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Project  string `form:"project"`
}
