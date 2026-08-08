package router

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
	"github.com/watch-tower-org/watchdog/backend/pkg"
	"github.com/watch-tower-org/watchdog/backend/pkg/alert_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/api_keys"
	"github.com/watch-tower-org/watchdog/backend/pkg/auth"
	"github.com/watch-tower-org/watchdog/backend/pkg/dashboard"
	"github.com/watch-tower-org/watchdog/backend/pkg/email_settings"
	"github.com/watch-tower-org/watchdog/backend/pkg/events"
	"github.com/watch-tower-org/watchdog/backend/pkg/ingestion"
	"github.com/watch-tower-org/watchdog/backend/pkg/issues"
	"github.com/watch-tower-org/watchdog/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchdog/backend/pkg/settings"
	"github.com/watch-tower-org/watchdog/backend/web"
)

func AppRouter(app *pkg.Application) (*gin.Engine, error) {
	gin.SetMode(app.Config.Server.Mode)
	router := gin.New()

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.RemoveExtraSlash = false

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(&app.Config.CORS))
	router.Use(middleware.RateLimit(&app.Config.RateLimit))
	router.Use(middleware.SecurityHeaders())

	v1 := router.Group("/api/watchtower/v1")
	{
		auth.Router(v1, app.AuthC, &app.Config.JWT)
		settings.Router(v1, app.SettingsC, &app.Config.JWT)
		email_settings.Router(v1, app.EmailSettingsC, &app.Config.JWT)
		alert_settings.Router(v1, app.AlertSettingsC, &app.Config.JWT)
		api_keys.Router(v1, app.ApiKeysC, &app.Config.JWT)
		recipient_lists.Router(v1, app.RecipientListsC, &app.Config.JWT)
		dashboard.Router(v1, app.DashboardC, &app.Config.JWT)
		ingestion.Router(v1, app.IngestionC, app.DB)
		issues.Router(v1, app.IssuesC, &app.Config.JWT)
		events.Router(v1, app.EventsC, &app.Config.JWT)
	}

	web.Register(router, "/api/watchtower/v1")

	return router, nil
}
