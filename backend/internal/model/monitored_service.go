package model

import (
	"time"

	"github.com/uptrace/bun"
)

// ServiceStatus is the last status of an uptime-monitored service.
type ServiceStatus string

const (
	ServiceStatusUp   ServiceStatus = "up"
	ServiceStatusDown ServiceStatus = "down"
)

type MonitoredService struct {
	bun.BaseModel `bun:"table:monitored_services,alias:ms"`

	ID                  int64          `bun:"id,pk,autoincrement" json:"id"`
	Name                string         `bun:"name,notnull" json:"name"`
	URL                 string         `bun:"url,notnull" json:"url"`
	IntervalSeconds     int            `bun:"interval_seconds,notnull,default:60" json:"interval_seconds"`
	TimeoutSeconds      int            `bun:"timeout_seconds,notnull,default:10" json:"timeout_seconds"`
	FailuresBeforeAlert int            `bun:"failures_before_alert,notnull,default:1" json:"failures_before_alert"`
	RecipientListID     *int64         `bun:"recipient_list_id" json:"recipient_list_id"`
	RecipientList       *RecipientList `bun:"rel:belongs-to,join:recipient_list_id=id" json:"recipient_list,omitempty"`
	IsActive            bool           `bun:"is_active,notnull,default:true" json:"is_active"`
	LastStatus          *ServiceStatus `bun:"last_status" json:"last_status"`
	LastCheckedAt       *time.Time     `bun:"last_checked_at" json:"last_checked_at"`
	LastUpAt            *time.Time     `bun:"last_up_at" json:"last_up_at"`
	LastDownAt          *time.Time     `bun:"last_down_at" json:"last_down_at"`
	ConsecutiveFailures int            `bun:"consecutive_failures,notnull,default:0" json:"consecutive_failures"`
	CreatedAt           time.Time      `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt           time.Time      `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type CreateMonitoredServiceRequest struct {
	Name                string `json:"name" validate:"required"`
	URL                 string `json:"url" validate:"required"`
	IntervalSeconds     *int   `json:"interval_seconds"`
	TimeoutSeconds      *int   `json:"timeout_seconds"`
	FailuresBeforeAlert *int   `json:"failures_before_alert"`
	RecipientListID     *int64 `json:"recipient_list_id"`
	IsActive            *bool  `json:"is_active"`
}

// UpdateMonitoredServiceRequest uses pointer fields for partial updates.
// RecipientListID is a pointer-to-pointer: a non-nil inner pointer sets a new
// list, a nil inner pointer (field present as json:null) clears the list, and
// an absent field leaves it unchanged.
type UpdateMonitoredServiceRequest struct {
	Name                *string `json:"name"`
	URL                 *string `json:"url"`
	IntervalSeconds     *int    `json:"interval_seconds"`
	TimeoutSeconds      *int    `json:"timeout_seconds"`
	FailuresBeforeAlert *int    `json:"failures_before_alert"`
	RecipientListID     **int64 `json:"recipient_list_id"`
	IsActive            *bool   `json:"is_active"`
}

type ListMonitoredServicesRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
}
