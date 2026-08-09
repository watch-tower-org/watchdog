package dashboard

import (
	"context"
	"errors"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

type Summary struct {
	RecipientLists int64 `json:"recipient_lists"`
	ApiKeys        int64 `json:"api_keys"`
	Issues         int64 `json:"issues"`
	Events         int64 `json:"events"`
	Alerts         int64 `json:"alerts"`
}

func (c *Controller) GetSummary(ctx context.Context) (*Summary, error) {
	summary := &Summary{}

	counts := []struct {
		table string
		dst   *int64
	}{
		{"recipient_lists", &summary.RecipientLists},
		{"api_keys", &summary.ApiKeys},
		{"issues", &summary.Issues},
		{"events", &summary.Events},
		{"alert_log", &summary.Alerts},
	}

	for _, cnt := range counts {
		err := c.db.NewRaw("SELECT COUNT(*) FROM "+cnt.table).Scan(ctx, cnt.dst)
		if err != nil {
			logger.Ctx(ctx).Error().Msgf("failed to count %s: %v", cnt.table, err)
			return nil, errors.New("An unexpected error occurred. Please try again.")
		}
	}

	return summary, nil
}
