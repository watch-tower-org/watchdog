package pkg

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/cache"
	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/database"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
	"github.com/watch-tower-org/watchtower/backend/pkg/alert_log"
	"github.com/watch-tower-org/watchtower/backend/pkg/alert_rules"
	"github.com/watch-tower-org/watchtower/backend/pkg/alert_settings"
	"github.com/watch-tower-org/watchtower/backend/pkg/api_keys"
	"github.com/watch-tower-org/watchtower/backend/pkg/auth"
	"github.com/watch-tower-org/watchtower/backend/pkg/dashboard"
	"github.com/watch-tower-org/watchtower/backend/pkg/email_settings"
	"github.com/watch-tower-org/watchtower/backend/pkg/events"
	"github.com/watch-tower-org/watchtower/backend/pkg/ingestion"
	"github.com/watch-tower-org/watchtower/backend/pkg/issues"
	"github.com/watch-tower-org/watchtower/backend/pkg/notifier"
	"github.com/watch-tower-org/watchtower/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchtower/backend/pkg/selfreport"
	"github.com/watch-tower-org/watchtower/backend/pkg/settings"
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
	SelfReport      *selfreport.Reporter
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
	app.initSelfReport()
	app.Notifier.Start()

	return app, nil
}

func (app *Application) initControllers(db *database.Database) {
	alertSettingsCache := cache.NewSingleton[model.AlertSettings]()
	emailSettingsCache := cache.NewSingleton[model.EmailSettings]()
	alertRulesCache := cache.NewMap[int64, model.AlertRule]()
	recipientListsCache := cache.NewMap[int64, model.RecipientList]()

	app.Notifier = notifier.NewNotifier(db.DB, 0, &notifier.Caches{
		AlertSettings: alertSettingsCache,
		EmailSettings: emailSettingsCache,
		Rules:         alertRulesCache,
		Recipients:    recipientListsCache,
	})
	app.SettingsC = settings.NewController(db.DB)
	app.EmailSettingsC = email_settings.NewController(db.DB, emailSettingsCache)
	app.AlertSettingsC = alert_settings.NewController(db.DB, alertSettingsCache)
	app.ApiKeysC = api_keys.NewController(db.DB)
	app.RecipientListsC = recipient_lists.NewController(db.DB, recipientListsCache)
	app.AuthC = auth.NewController(db.DB, &app.Config.JWT)
	app.DashboardC = dashboard.NewController(db.DB)
	app.IngestionC = ingestion.NewController(db.DB, app.Notifier)
	app.IssuesC = issues.NewController(db.DB)
	app.EventsC = events.NewController(db.DB)
	app.AlertRulesC = alert_rules.NewController(db.DB, alertRulesCache)
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
	if err := app.AuthC.EnsureAdmin(app.Config.Admin.Username, app.Config.Admin.Password, app.Config.Admin.ResetPassword); err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}

	logger.Info().Msgf("Default admin account ready for user: %s", app.Config.Admin.Username)

	return nil
}

// initSelfReport wires the dogfooding reporter. Failures are non-fatal: the
// backend keeps running with stderr/log-file visibility.
func (app *Application) initSelfReport() {
	sr, err := selfreport.New(
		app.Config.SelfReport,
		app.IngestionC,
	)
	if err != nil {
		logger.Warn().Err(err).Msg("self-reporting disabled")
		return
	}
	app.SelfReport = sr
}

func (app *Application) Shutdown() {
	if app.SelfReport != nil {
		app.SelfReport.Close()
	}
	if app.Notifier != nil {
		app.Notifier.Shutdown()
	}
}
