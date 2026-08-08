package api_keys

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
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	keys, pageInfo, err := h.controller.List(c.Request.Context(), page, pageSize)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "api keys retrieved successfully", keys, pageInfo)
}

func (h *Handler) Create(c *gin.Context) {
	var req model.CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	created, err := h.controller.Create(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Created(c, "api key created successfully", created)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	ak, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "api key retrieved successfully", ak)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	var req model.UpdateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	ak, err := h.controller.Update(c.Request.Context(), id, &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "api key updated successfully", ak)
}

func (h *Handler) Revoke(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	if err := h.controller.Revoke(c.Request.Context(), id); err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "api key revoked successfully", nil)
}
