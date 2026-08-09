package issues

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) List(c *gin.Context) {
	var req model.ListIssuesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.BadRequest(c, "Invalid query parameters")
		return
	}

	issues, pageInfo, err := h.controller.List(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "issues retrieved successfully", issues, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	issue, events, err := h.controller.GetWithEvents(c.Request.Context(), id, 0)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "issue retrieved successfully", map[string]interface{}{
		"issue":  issue,
		"events": events,
	})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	var req model.UpdateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	if req.Status == nil {
		res.BadRequest(c, "status is required")
		return
	}

	issue, err := h.controller.UpdateStatus(c.Request.Context(), id, *req.Status)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "issue updated successfully", issue)
}
