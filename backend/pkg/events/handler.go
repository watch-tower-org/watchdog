package events

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
	var req model.ListEventsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.BadRequest(c, "Invalid query parameters")
		return
	}

	events, pageInfo, err := h.controller.List(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "events retrieved successfully", events, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	event, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "event retrieved successfully", event)
}
