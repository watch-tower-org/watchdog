package version

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/version"
)

type Controller struct{}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Version(ctx *gin.Context) {
	res.Ok(ctx, "version retrieved", map[string]any{
		"product": "watchtower",
		"version": version.Version,
	})
}
