package httpx

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents the standard API response structure
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK sends a successful response with default message "success"
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// OKMsg sends a successful response with custom message
func OKMsg(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Fail sends an error response with specified HTTP status, business code, and message
func Fail(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// FailErr sends an error response from an AppError
// If AppError.Err is not nil, it will be logged but not returned to client
func FailErr(c *gin.Context, err *AppError) {
	// Log internal error if present (for debugging, not returned to client)
	if err.Err != nil {
		log.Printf("[ERROR] %s (code=%d, internal_err=%v)", err.Message, err.Code, err.Err)
	}

	// Use err.Data if present, otherwise nil
	data := err.Data
	if data == nil {
		data = nil // Explicitly set to nil for JSON serialization
	}

	c.JSON(err.HTTPStatus, Response{
		Code:    err.Code,
		Message: err.Message,
		Data:    data,
	})
}
