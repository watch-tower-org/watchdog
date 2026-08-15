package uptime

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
)

type Handler struct {
	controller *Controller
}

// ServiceDetail is the GET /:id payload: the service plus its recent checks.
type ServiceDetail struct {
	model.MonitoredService
	RecentChecks []model.UptimeCheck `json:"recent_checks"`
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	search := c.Query("search")

	services, pageInfo, err := h.controller.List(c.Request.Context(), search, page, pageSize)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "monitored services retrieved successfully", services, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	svc, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			res.NotFound(c, "monitored service not found")
			return
		}
		res.BadRequest(c, err.Error())
		return
	}

	checks, err := h.controller.RecentChecks(c.Request.Context(), id, defaultRecentChecks)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "monitored service retrieved successfully", ServiceDetail{
		MonitoredService: *svc,
		RecentChecks:     checks,
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req model.CreateMonitoredServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	svc, err := h.controller.Create(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Created(c, "monitored service created successfully", svc)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	var req model.UpdateMonitoredServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	svc, err := h.controller.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			res.NotFound(c, "monitored service not found")
			return
		}
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "monitored service updated successfully", svc)
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

	res.NoContent(c, "monitored service deleted successfully")
}

func (h *Handler) CheckNow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	result, err := h.controller.CheckNow(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			res.NotFound(c, "monitored service not found")
			return
		}
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "check completed", result)
}
