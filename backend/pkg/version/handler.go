package version

import (
	"github.com/gin-gonic/gin"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

func (h *Handler) Version(c *gin.Context) {
	h.controller.Version(c)
}