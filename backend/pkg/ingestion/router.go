package ingestion

import (
	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, db *bun.DB, rateLimitCfg *config.RateLimitConfig) {
	h := NewHandler(controller)

	grp := r.Group("/events")
	grp.Use(middleware.ApiKeyAuth(db))
	grp.Use(middleware.ApiKeyRateLimit(rateLimitCfg))

	grp.POST("", h.Ingest)
}
