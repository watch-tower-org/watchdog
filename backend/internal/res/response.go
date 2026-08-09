package res

import (
	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SuccessPaginationResponse struct {
	Success  bool           `json:"success"`
	Code     int            `json:"code"`
	Message  string         `json:"message"`
	Data     interface{}    `json:"data,omitempty"`
	PageInfo model.PageInfo `json:"page_info"`
}

type ErrorsResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Error   string `json:"error,omitempty"`
}

func successResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	response := SuccessResponse{
		Success: true,
		Code:    statusCode,
		Message: message,
		Data:    data,
	}
	c.JSON(statusCode, response)
}

func errorResponse(c *gin.Context, statusCode int, err string) {
	response := ErrorsResponse{
		Success: false,
		Code:    statusCode,
		Error:   err,
	}
	c.JSON(statusCode, response)
}

func OkPagination(c *gin.Context, message string, data interface{}, pageInfo *model.PageInfo) {
	response := SuccessPaginationResponse{
		Success:  true,
		Code:     200,
		Message:  message,
		Data:     data,
		PageInfo: *pageInfo,
	}
	c.JSON(200, response)
}

func Ok(c *gin.Context, message string, data interface{}) {
	successResponse(c, 200, message, data)
}

func Created(c *gin.Context, message string, data interface{}) {
	successResponse(c, 201, message, data)
}

func NoContent(c *gin.Context, message string) {
	successResponse(c, 204, message, nil)
}

func BadRequest(c *gin.Context, err string) {
	errorResponse(c, 400, err)
}

func Unauthorized(c *gin.Context, err string) {
	errorResponse(c, 401, err)
}

func Forbidden(c *gin.Context, err string) {
	errorResponse(c, 403, err)
}

func NotFound(c *gin.Context, err string) {
	errorResponse(c, 404, err)
}

func Conflict(c *gin.Context, err string) {
	errorResponse(c, 409, err)
}

func TooManyRequests(c *gin.Context, err string) {
	errorResponse(c, 429, err)
}

func InternalServerError(c *gin.Context, err string) {
	errorResponse(c, 500, err)
}
