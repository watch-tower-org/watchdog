package router

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
	"github.com/watch-tower-org/watchtower/backend/pkg"
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
	"github.com/watch-tower-org/watchtower/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchtower/backend/pkg/settings"
	"github.com/watch-tower-org/watchtower/backend/pkg/version"
	"github.com/watch-tower-org/watchtower/backend/web"
)

func AppRouter(app *pkg.Application) (*gin.Engine, error) {
	gin.SetMode(app.Config.Server.Mode)
	router := gin.New()

	// Trust only explicitly configured proxy CIDRs so X-Forwarded-For can't be
	// spoofed to reset rate limits or fake client IPs in logs. With no proxies
	// configured, ClientIP is the direct remote address.
	if err := router.SetTrustedProxies(app.Config.Server.TrustedProxies); err != nil {
		return nil, err
	}

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.RemoveExtraSlash = false

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery(func(v any, ctx map[string]any) {
		if app.SelfReport != nil {
			app.SelfReport.Recover(v, ctx)
		}
	}))
	router.Use(middleware.CORS(&app.Config.CORS))
	router.Use(middleware.RateLimit(&app.Config.RateLimit))
	router.Use(middleware.SecurityHeaders())

	v1 := router.Group("/api/watchtower/v1")
	{
		auth.Router(v1, app.AuthC, &app.Config.JWT, app.Config.Server.CookieSecure)
		settings.Router(v1, app.SettingsC, &app.Config.JWT)
		email_settings.Router(v1, app.EmailSettingsC, &app.Config.JWT)
		alert_settings.Router(v1, app.AlertSettingsC, &app.Config.JWT)
		api_keys.Router(v1, app.ApiKeysC, &app.Config.JWT)
		recipient_lists.Router(v1, app.RecipientListsC, &app.Config.JWT)
		dashboard.Router(v1, app.DashboardC, &app.Config.JWT)
		ingestion.Router(v1, app.IngestionC, app.DB, &app.Config.RateLimit)
		issues.Router(v1, app.IssuesC, &app.Config.JWT)
		events.Router(v1, app.EventsC, &app.Config.JWT)
		alert_rules.Router(v1, app.AlertRulesC, &app.Config.JWT)
		alert_log.Router(v1, app.AlertLogC, &app.Config.JWT)
		version.Router(v1, app.DB)
	}

	web.Register(router, "/api/watchtower/v1")

	return router, nil
}
