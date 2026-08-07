package alert_settings

import (
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

func (h *Handler) Get(c *gin.Context) {
	s, err := h.controller.Get(c.Request.Context())
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert settings retrieved successfully", s)
}

func (h *Handler) Update(c *gin.Context) {
	var req model.UpdateAlertSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	updated, err := h.controller.Update(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "alert settings updated successfully", updated)
}
