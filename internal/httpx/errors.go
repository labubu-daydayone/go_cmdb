package httpx

import (
	"fmt"
	"net/http"
)

// Business error codes
const (
	// Success
	CodeSuccess = 0

	// Authentication/Authorization errors (1000-1099)
	CodeUnauthorized   = 1001 // Not logged in / Token missing
	CodeInvalidToken   = 1002 // Token invalid
	CodeTokenExpired   = 1003 // Token expired
	CodeForbidden      = 1004 // No permission

	// Parameter errors (2000-2099)
	CodeParamMissing  = 2001 // Parameter missing
	CodeParamInvalid  = 2002 // Parameter format error
	CodeParamIllegal  = 2003 // Parameter value illegal

	// Resource/Business errors (3000-3999)
	CodeNotFound      = 3001 // Resource not found
	CodeAlreadyExists = 3002 // Resource already exists
	CodeStateConflict = 3003 // Current state does not allow operation

	// System errors (5000-5999)
	CodeInternalError = 5001 // Internal service error
	CodeDatabaseError = 5002 // Database error
	CodeExternalError = 5003 // External dependency failure
)

// AppError represents an application error with HTTP status and business code
type AppError struct {
	HTTPStatus int         // HTTP status code
	Code       int         // Business error code
	Message    string      // User-facing error message
	Err        error       // Internal error (for logging only, not returned to client)
	Data       interface{} // Additional data (for detailed error information)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code=%d, message=%s, err=%v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// WithData adds additional data to the error
func (e *AppError) WithData(data interface{}) *AppError {
	e.Data = data
	return e
}

// NewAppError creates a new AppError
func NewAppError(httpStatus, code int, message string, err error) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Err:        err,
	}
}

// Authentication/Authorization error constructors

// ErrUnauthorized creates a 401 unauthorized error
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "unauthorized"
	}
	return NewAppError(http.StatusUnauthorized, CodeUnauthorized, message, nil)
}

// ErrInvalidToken creates a 401 invalid token error
func ErrInvalidToken(message string) *AppError {
	if message == "" {
		message = "invalid token"
	}
	return NewAppError(http.StatusUnauthorized, CodeInvalidToken, message, nil)
}

// ErrTokenExpired creates a 401 token expired error
func ErrTokenExpired(message string) *AppError {
	if message == "" {
		message = "token expired"
	}
	return NewAppError(http.StatusUnauthorized, CodeTokenExpired, message, nil)
}

// ErrForbidden creates a 403 forbidden error
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "forbidden"
	}
	return NewAppError(http.StatusForbidden, CodeForbidden, message, nil)
}

// Parameter error constructors

// ErrParamMissing creates a 400 parameter missing error
func ErrParamMissing(message string) *AppError {
	if message == "" {
		message = "parameter missing"
	}
	return NewAppError(http.StatusBadRequest, CodeParamMissing, message, nil)
}

// ErrParamInvalid creates a 400 parameter invalid error
func ErrParamInvalid(message string) *AppError {
	if message == "" {
		message = "parameter format error"
	}
	return NewAppError(http.StatusBadRequest, CodeParamInvalid, message, nil)
}

// ErrParamIllegal creates a 400 parameter illegal error
func ErrParamIllegal(message string) *AppError {
	if message == "" {
		message = "parameter value illegal"
	}
	return NewAppError(http.StatusBadRequest, CodeParamIllegal, message, nil)
}

// Resource/Business error constructors

// ErrNotFound creates a 404 not found error
func ErrNotFound(message string) *AppError {
	if message == "" {
		message = "resource not found"
	}
	return NewAppError(http.StatusNotFound, CodeNotFound, message, nil)
}

// ErrAlreadyExists creates a 409 already exists error
func ErrAlreadyExists(message string) *AppError {
	if message == "" {
		message = "resource already exists"
	}
	return NewAppError(http.StatusConflict, CodeAlreadyExists, message, nil)
}

// ErrStateConflict creates a 409 state conflict error
func ErrStateConflict(message string) *AppError {
	if message == "" {
		message = "current state does not allow operation"
	}
	return NewAppError(http.StatusConflict, CodeStateConflict, message, nil)
}

// System error constructors

// ErrInternalError creates a 500 internal error
func ErrInternalError(message string, err error) *AppError {
	if message == "" {
		message = "internal error"
	}
	return NewAppError(http.StatusInternalServerError, CodeInternalError, message, err)
}

// ErrDatabaseError creates a 500 database error
func ErrDatabaseError(message string, err error) *AppError {
	if message == "" {
		message = "database error"
	}
	return NewAppError(http.StatusInternalServerError, CodeDatabaseError, message, err)
}

// ErrExternalError creates a 502 external dependency error
func ErrExternalError(message string, err error) *AppError {
	if message == "" {
		message = "external dependency failure"
	}
	return NewAppError(http.StatusBadGateway, CodeExternalError, message, err)
}
