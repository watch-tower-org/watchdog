package version

import (
	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/middleware"
)

func Router(r *gin.RouterGroup, db *bun.DB) {
	h := NewHandler(NewController())

	grp := r.Group("")
	grp.Use(middleware.ApiKeyAuth(db))
	grp.GET("/version", h.Version)
}