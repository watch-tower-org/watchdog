package email_settings

import (
	"net/http"

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

func (h *Handler) Get(c *gin.Context) {
	s, err := h.controller.Get(c.Request.Context())
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "email settings retrieved successfully", s)
}

func (h *Handler) Update(c *gin.Context) {
	var req model.UpdateEmailSettingsRequest
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

	res.Ok(c, "email settings updated successfully", updated)
}

func (h *Handler) TestEmail(c *gin.Context) {
	var req model.TestEmailRequest

	to := c.Query("to")
	if to != "" {
		req.To = to
	}

	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			res.BadRequest(c, "Invalid request body")
			return
		}
	} else if to == "" {
		res.BadRequest(c, "Recipient email address is required")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	err := h.controller.TestSMTP(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "test email sent successfully", nil)
}
