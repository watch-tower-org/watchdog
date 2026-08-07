package router

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
	"github.com/watch-tower-org/watchdog/backend/internal/res"
	"github.com/watch-tower-org/watchdog/backend/pkg"
	"github.com/watch-tower-org/watchdog/backend/pkg/auth"
	"github.com/watch-tower-org/watchdog/backend/pkg/recipient_lists"
	"github.com/watch-tower-org/watchdog/backend/pkg/settings"
)

func AppRouter(app *pkg.Application) (*gin.Engine, error) {
	gin.SetMode(app.Config.Server.Mode)
	router := gin.New()

	router.RedirectTrailingSlash = true
	router.RedirectFixedPath = true
	router.RemoveExtraSlash = true

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(&app.Config.CORS))
	router.Use(middleware.RateLimit(&app.Config.RateLimit))
	router.Use(middleware.SecurityHeaders())

	router.NoRoute(func(c *gin.Context) {
		res.NotFound(c, "The requested resource was not found")
	})

	v1 := router.Group("/api/watchtower/v1")
	{
		auth.Router(v1, app.AuthC, &app.Config.JWT)
		settings.Router(v1, app.SettingsC, &app.Config.JWT)
		recipient_lists.Router(v1, app.RecipientListsC, &app.Config.JWT)
	}

	return router, nil
}
