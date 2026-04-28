package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Code: 0, Message: "created", Data: data})
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{Code: 400, Message: msg})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{Code: 401, Message: "unauthorized"})
}

func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{Code: 403, Message: "forbidden"})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{Code: 404, Message: msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, Response{Code: 409, Message: msg})
}

func TooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, Response{Code: 429, Message: "too many requests"})
}

func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: msg})
}

func Error(c *gin.Context, httpStatus int, code, msg string) {
	c.JSON(httpStatus, Response{Code: httpStatus, Message: msg})
}
