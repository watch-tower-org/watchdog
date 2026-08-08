package database

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

func AutoMigration(db *bun.DB, ctx context.Context) error {
	db.RegisterModel((*model.Settings)(nil))
	db.RegisterModel((*model.EmailSettings)(nil))
	db.RegisterModel((*model.AlertSettings)(nil))
	db.RegisterModel((*model.ApiKey)(nil))
	db.RegisterModel((*model.Admin)(nil))
	db.RegisterModel((*model.RecipientList)(nil))
	db.RegisterModel((*model.AlertRule)(nil))
	db.RegisterModel((*model.Issue)(nil))
	db.RegisterModel((*model.Event)(nil))
	db.RegisterModel((*model.AlertLog)(nil))

	models := []interface{}{
		(*model.Settings)(nil),
		(*model.EmailSettings)(nil),
		(*model.AlertSettings)(nil),
		(*model.ApiKey)(nil),
		(*model.Admin)(nil),
		(*model.RecipientList)(nil),
		(*model.AlertRule)(nil),
		(*model.Issue)(nil),
		(*model.Event)(nil),
		(*model.AlertLog)(nil),
	}

	for _, i := range models {
		_, err := db.NewCreateTable().Model(i).IfNotExists().Exec(ctx)
		if err != nil {
			logger.Error().Msgf("Failed to create table for model: %T", i)
			logger.Error().Msgf("Error: %v", err)
			return err
		}
	}

	// Index supporting issue-event lookups.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_issue_id ON events (issue_id)`); err != nil {
		logger.Error().Msgf("failed to create events issue_id index: %v", err)
		return err
	}

	// Index supporting event listing filtered by project.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_project ON events (project)`); err != nil {
		logger.Error().Msgf("failed to create events project index: %v", err)
		return err
	}

	// Index supporting issue listing filtered by project/status.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_issues_project_status ON issues (project, status)`); err != nil {
		logger.Error().Msgf("failed to create issues project/status index: %v", err)
		return err
	}

	// Index supporting alert log lookups.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_alert_log_issue_id ON alert_log (issue_id)`); err != nil {
		logger.Error().Msgf("failed to create alert_log issue_id index: %v", err)
		return err
	}

	logger.Info().Msgf("Database tables created successfully")

	return nil
}
