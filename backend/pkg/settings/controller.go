package settings

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) loadSettings(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	err := c.db.NewSelect().
		Model(&s).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("Error getting settings: %v", err)
		return nil, errors.New("Failed to retrieve settings.")
	}

	return &s, nil
}

func (c *Controller) Get(ctx context.Context) (*model.Settings, error) {
	return c.loadSettings(ctx)
}

func (c *Controller) Update(ctx context.Context, req *model.UpdateSettingsRequest) (*model.Settings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	if req.SetupComplete != nil {
		s.SetupComplete = *req.SetupComplete
	}

	s.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(s).
		Where("id = ?", s.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update settings: %v", err)
		return nil, errors.New("Failed to update settings.")
	}

	return s, nil
}

func (c *Controller) CreateDefaultSettings() error {
	ctx := context.Background()

	nb, err := c.db.NewSelect().
		Model((*model.Settings)(nil)).
		Count(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("count settings: %v", err)
		return err
	}

	if nb != 0 {
		return nil
	}

	defaultSettings := &model.Settings{
		SetupComplete: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = c.db.NewInsert().
		Model(defaultSettings).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create default settings: %v", err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	return nil
}
