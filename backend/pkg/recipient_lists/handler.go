package recipient_lists

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
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	search := c.Query("search")

	lists, pageInfo, err := h.controller.List(c.Request.Context(), search, page, pageSize)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.OkPagination(c, "recipient lists retrieved successfully", lists, pageInfo)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	rl, err := h.controller.GetByID(c.Request.Context(), id)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "recipient list retrieved successfully", rl)
}

func (h *Handler) Create(c *gin.Context) {
	var req model.CreateRecipientListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	rl, err := h.controller.Create(c.Request.Context(), &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Created(c, "recipient list created successfully", rl)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		res.BadRequest(c, "Invalid id")
		return
	}

	var req model.UpdateRecipientListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.BadRequest(c, "Invalid request body")
		return
	}

	if err := validator.Validate(&req); err != nil {
		res.BadRequest(c, validator.ValidationError(err))
		return
	}

	rl, err := h.controller.Update(c.Request.Context(), id, &req)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Ok(c, "recipient list updated successfully", rl)
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

	res.NoContent(c, "recipient list deleted successfully")
}
