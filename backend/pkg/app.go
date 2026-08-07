package pkg

import (
	"fmt"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/database"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/validator"
	"github.com/watch-tower-org/watchdog/backend/pkg/auth"
	"github.com/watch-tower-org/watchdog/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchdog/backend/pkg/settings"
)

type Application struct {
	Config *config.Config

	SettingsC       *settings.Controller
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
	app.RecipientListsC = recipient_lists.NewController(db.DB)
	app.AuthC = auth.NewController(db.DB, &app.Config.JWT)
}

func (app *Application) initDefaults() error {
	adminPwd, err := app.AuthC.HashPassword(app.Config.Admin.Password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	if err := app.SettingsC.CreateDefaultSettings(app.Config.Admin.Username, adminPwd); err != nil {
		return fmt.Errorf("create default settings: %w", err)
	}

	logger.Info().Msgf("Default admin account ready for user: %s", app.Config.Admin.Username)

	return nil
}

func (app *Application) Shutdown() {}
