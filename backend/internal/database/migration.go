package database

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
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

	// Index supporting spike window COUNT queries (per rule per event).
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp)`); err != nil {
		logger.Error().Msgf("failed to create events timestamp index: %v", err)
		return err
	}

	// Index supporting dashboard trend range scans on server receive time.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_created_at ON events (created_at)`); err != nil {
		logger.Error().Msgf("failed to create events created_at index: %v", err)
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

	// Index supporting refresh-session lookups and cleanup by admin.
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_auth_sessions_admin_id ON auth_sessions (admin_id)`); err != nil {
		logger.Error().Msgf("failed to create auth_sessions admin_id index: %v", err)
		return err
	}

	// Add the masked column to api_keys if it doesn't exist yet (older installs).
	if _, err := db.ExecContext(ctx, `ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS masked text NOT NULL DEFAULT ''`); err != nil {
		logger.Error().Msgf("failed to add masked column to api_keys: %v", err)
		return err
	}

	// Add the message column to events if it doesn't exist yet (older installs).
	if _, err := db.ExecContext(ctx, `ALTER TABLE events ADD COLUMN IF NOT EXISTS message text`); err != nil {
		logger.Error().Msgf("failed to add message column to events: %v", err)
		return err
	}

	// throttle_window on alert_rules becomes nullable: NULL (or legacy 0) means
	// "use the global throttle setting". Older installs created the column
	// NOT NULL DEFAULT 60, so relax it and normalize stored 0s.
	if _, err := db.ExecContext(ctx, `ALTER TABLE alert_rules ALTER COLUMN throttle_window DROP NOT NULL`); err != nil {
		logger.Error().Msgf("failed to drop NOT NULL on alert_rules.throttle_window: %v", err)
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE alert_rules SET throttle_window = NULL WHERE throttle_window = 0`); err != nil {
		logger.Error().Msgf("failed to normalize alert_rules.throttle_window: %v", err)
		return err
	}

	logger.Info().Msgf("Database tables created successfully")

	return nil
}
