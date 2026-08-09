package alert_log

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) List(c *gin.Context) {
	var req model.ListAlertLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.BadRequest(c, "Invalid query parameters")
		return
	}

	logs, pageInfo, err := h.controller.List(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "alerts retrieved successfully", logs, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	log, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert retrieved successfully", log)
}
