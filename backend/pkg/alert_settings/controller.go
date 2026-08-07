package alert_settings

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) loadSettings(ctx context.Context) (*model.AlertSettings, error) {
	var s model.AlertSettings
	err := c.db.NewSelect().
		Model(&s).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("Error getting alert settings: %v", err)
		return nil, errors.New("Failed to retrieve alert settings.")
	}

	return &s, nil
}

func (c *Controller) Get(ctx context.Context) (*model.AlertSettings, error) {
	return c.loadSettings(ctx)
}

func (c *Controller) Update(ctx context.Context, req *model.UpdateAlertSettingsRequest) (*model.AlertSettings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	if req.ThrottleWindow != nil {
		s.ThrottleWindow = *req.ThrottleWindow
	}

	s.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(s).
		Where("id = ?", s.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update alert settings: %v", err)
		return nil, errors.New("Failed to update alert settings.")
	}

	return s, nil
}

func (c *Controller) CreateDefaultSettings() error {
	ctx := context.Background()

	nb, err := c.db.NewSelect().
		Model((*model.AlertSettings)(nil)).
		Count(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("count alert settings: %v", err)
		return err
	}

	if nb != 0 {
		return nil
	}

	defaultSettings := &model.AlertSettings{
		ThrottleWindow: 60,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = c.db.NewInsert().
		Model(defaultSettings).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create default alert settings: %v", err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	return nil
}
