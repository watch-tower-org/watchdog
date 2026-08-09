package dashboard

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) Summary(c *gin.Context) {
	summary, err := h.controller.GetSummary(c.Request.Context())
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "dashboard summary retrieved successfully", summary)
}

func (h *Handler) Trends(c *gin.Context) {
	rangeKey := c.DefaultQuery("range", "7d")
	trends, err := h.controller.GetTrends(c.Request.Context(), rangeKey)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "dashboard trends retrieved successfully", trends)
}
