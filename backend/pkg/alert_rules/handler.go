package alert_rules

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchdog/backend/internal/model"
	"github.com/watch-tower-org/watchdog/backend/internal/res"
	"github.com/watch-tower-org/watchdog/backend/internal/validator"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) List(c *gin.Context) {
	var req model.ListAlertRulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.BadRequest(c, "Invalid query parameters")
		return
	}

	rules, pageInfo, err := h.controller.List(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "alert rules retrieved successfully", rules, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	rule, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert rule retrieved successfully", rule)
}

func (h *Handler) Create(c *gin.Context) {
	var req model.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	rule, err := h.controller.Create(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Created(c, "alert rule created successfully", rule)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	var req model.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	rule, err := h.controller.Update(c.Request.Context(), id, &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert rule updated successfully", rule)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	if err := h.controller.Delete(c.Request.Context(), id); err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert rule deleted successfully", nil)
}
