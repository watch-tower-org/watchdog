package pkg

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/database"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/validator"
	"github.com/watch-tower-org/watchdog/backend/pkg/alert_log"
	"github.com/watch-tower-org/watchdog/backend/pkg/alert_rules"
	"github.com/watch-tower-org/watchdog/backend/pkg/alert_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/api_keys"
	"github.com/watch-tower-org/watchdog/backend/pkg/auth"
	"github.com/watch-tower-org/watchdog/backend/pkg/dashboard"
	"github.com/watch-tower-org/watchdog/backend/pkg/email_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/events"
	"github.com/watch-tower-org/watchdog/backend/pkg/ingestion"
	"github.com/watch-tower-org/watchdog/backend/pkg/issues"
	"github.com/watch-tower-org/watchdog/backend/pkg/notifier"
	"github.com/watch-tower-org/watchdog/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchdog/backend/pkg/settings"
)

type Application struct {
	Config *config.Config
	DB     *bun.DB

	SettingsC       *settings.Controller
	EmailSettingsC  *email_settings.Controller
	AlertSettingsC  *alert_settings.Controller
	ApiKeysC        *api_keys.Controller
	RecipientListsC *recipient_lists.Controller
	AuthC           *auth.Controller
	DashboardC      *dashboard.Controller
	IngestionC      *ingestion.Controller
	IssuesC         *issues.Controller
	EventsC         *events.Controller
	AlertRulesC     *alert_rules.Controller
	AlertLogC       *alert_log.Controller
	Notifier        *notifier.Notifier
}

func NewApplication(cfg *config.Config, db *database.Database) (*Application, error) {
	app := &Application{
		Config: cfg,
		DB:     db.DB,
	}

	validator.Init()
	app.initControllers(db)
	if err := app.initDefaults(); err != nil {
		return nil, fmt.Errorf("init defaults: %w", err)
	}
	app.Notifier.Start()

	return app, nil
}

func (app *Application) initControllers(db *database.Database) {
	app.Notifier = notifier.NewNotifier(db.DB, 0)
	app.SettingsC = settings.NewController(db.DB)
	app.EmailSettingsC = email_settings.NewController(db.DB)
	app.AlertSettingsC = alert_settings.NewController(db.DB)
	app.ApiKeysC = api_keys.NewController(db.DB)
	app.RecipientListsC = recipient_lists.NewController(db.DB)
	app.AuthC = auth.NewController(db.DB, &app.Config.JWT)
	app.DashboardC = dashboard.NewController(db.DB)
	app.IngestionC = ingestion.NewController(db.DB, app.Notifier)
	app.IssuesC = issues.NewController(db.DB)
	app.EventsC = events.NewController(db.DB)
	app.AlertRulesC = alert_rules.NewController(db.DB)
	app.AlertLogC = alert_log.NewController(db.DB)
}

func (app *Application) initDefaults() error {
	if err := app.SettingsC.CreateDefaultSettings(); err != nil {
		return fmt.Errorf("create default settings: %w", err)
	}
	if err := app.EmailSettingsC.CreateDefaultSettings(); err != nil {
		return fmt.Errorf("create default email settings: %w", err)
	}
	if err := app.AlertSettingsC.CreateDefaultSettings(); err != nil {
		return fmt.Errorf("create default alert settings: %w", err)
	}
	if err := app.AuthC.EnsureAdmin(app.Config.Admin.Username, app.Config.Admin.Password); err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}

	logger.Info().Msgf("Default admin account ready for user: %s", app.Config.Admin.Username)

	return nil
}

func (app *Application) Shutdown() {
	if app.Notifier != nil {
		app.Notifier.Shutdown()
	}
}
