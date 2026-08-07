package dashboard

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/res"
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
