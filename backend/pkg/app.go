package pkg

import (
	"fmt"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/database"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/validator"
	"github.com/watch-tower-org/watchdog/backend/pkg/alert_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/api_keys"
	"github.com/watch-tower-org/watchdog/backend/pkg/auth"
	"github.com/watch-tower-org/watchdog/backend/pkg/email_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchdog/backend/pkg/settings"
)

type Application struct {
	Config *config.Config

	SettingsC       *settings.Controller
	EmailSettingsC  *email_settings.Controller
	AlertSettingsC  *alert_settings.Controller
	ApiKeysC        *api_keys.Controller
	RecipientListsC *recipient_lists.Controller
	AuthC           *auth.Controller
}

func NewApplication(cfg *config.Config, db *database.Database) (*Application, error) {
	app := &Application{
		Config: cfg,
	}

	validator.Init()
	app.initControllers(db)
	if err := app.initDefaults(); err != nil {
		return nil, fmt.Errorf("init defaults: %w", err)
	}

	return app, nil
}

func (app *Application) initControllers(db *database.Database) {
	app.SettingsC = settings.NewController(db.DB)
	app.EmailSettingsC = email_settings.NewController(db.DB)
	app.AlertSettingsC = alert_settings.NewController(db.DB)
	app.ApiKeysC = api_keys.NewController(db.DB)
	app.RecipientListsC = recipient_lists.NewController(db.DB)
	app.AuthC = auth.NewController(db.DB, &app.Config.JWT)
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

func (app *Application) Shutdown() {}
