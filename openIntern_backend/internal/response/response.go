package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess      = 0
	CodeBadRequest   = 1000
	CodeUnauthorized = 1001
	CodeForbidden    = 1003
	CodeNotFound     = 1004
	CodeInternal     = 1500
)

type Result struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func JSONSuccess(c *gin.Context, httpStatus int, data any) {
	c.JSON(httpStatus, Result{Code: CodeSuccess, Message: "success", Data: data})
}

func JSONMessage(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Result{Code: CodeSuccess, Message: message, Data: nil})
}

func JSONError(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Result{Code: code, Message: message, Data: nil})
}

func BadRequest(c *gin.Context) {
	JSONError(c, http.StatusBadRequest, CodeBadRequest, "bad request")
}

func Unauthorized(c *gin.Context) {
	JSONError(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
}

func Forbidden(c *gin.Context) {
	JSONError(c, http.StatusForbidden, CodeForbidden, "forbidden")
}

func NotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, CodeNotFound, message)
}

func InternalError(c *gin.Context) {
	JSONError(c, http.StatusInternalServerError, CodeInternal, "internal server error")
}
