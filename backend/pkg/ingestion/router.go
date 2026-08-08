package ingestion

import (
	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, controller *Controller, db *bun.DB) {
	h := NewHandler(controller)

	grp := r.Group("/events")
	grp.Use(middleware.ApiKeyAuth(db))

	grp.POST("", h.Ingest)
}
