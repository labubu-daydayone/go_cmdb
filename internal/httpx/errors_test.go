package httpx

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "error without internal err",
			err:  NewAppError(http.StatusBadRequest, CodeParamMissing, "param missing", nil),
			want: "code=2001, message=param missing",
		},
		{
			name: "error with internal err",
			err:  NewAppError(http.StatusInternalServerError, CodeInternalError, "internal error", errors.New("db connection failed")),
			want: "code=5001, message=internal error, err=db connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrUnauthorized(t *testing.T) {
	err := ErrUnauthorized("")
	if err.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusUnauthorized, err.HTTPStatus)
	}
	if err.Code != CodeUnauthorized {
		t.Errorf("Expected code %d, got %d", CodeUnauthorized, err.Code)
	}
	if err.Message != "unauthorized" {
		t.Errorf("Expected message 'unauthorized', got '%s'", err.Message)
	}
}

func TestErrParamMissing(t *testing.T) {
	err := ErrParamMissing("field 'name' is required")
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}
	if err.Code != CodeParamMissing {
		t.Errorf("Expected code %d, got %d", CodeParamMissing, err.Code)
	}
	if err.Message != "field 'name' is required" {
		t.Errorf("Expected custom message, got '%s'", err.Message)
	}
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound("user not found")
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusNotFound, err.HTTPStatus)
	}
	if err.Code != CodeNotFound {
		t.Errorf("Expected code %d, got %d", CodeNotFound, err.Code)
	}
}

func TestErrInternalError(t *testing.T) {
	internalErr := errors.New("database connection failed")
	err := ErrInternalError("internal error", internalErr)
	
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusInternalServerError, err.HTTPStatus)
	}
	if err.Code != CodeInternalError {
		t.Errorf("Expected code %d, got %d", CodeInternalError, err.Code)
	}
	if err.Err != internalErr {
		t.Errorf("Expected internal error to be preserved")
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
		min  int
		max  int
	}{
		{"CodeSuccess", CodeSuccess, 0, 0},
		{"CodeUnauthorized", CodeUnauthorized, 1000, 1099},
		{"CodeInvalidToken", CodeInvalidToken, 1000, 1099},
		{"CodeTokenExpired", CodeTokenExpired, 1000, 1099},
		{"CodeForbidden", CodeForbidden, 1000, 1099},
		{"CodeParamMissing", CodeParamMissing, 2000, 2099},
		{"CodeParamInvalid", CodeParamInvalid, 2000, 2099},
		{"CodeParamIllegal", CodeParamIllegal, 2000, 2099},
		{"CodeNotFound", CodeNotFound, 3000, 3999},
		{"CodeAlreadyExists", CodeAlreadyExists, 3000, 3999},
		{"CodeStateConflict", CodeStateConflict, 3000, 3999},
		{"CodeInternalError", CodeInternalError, 5000, 5999},
		{"CodeDatabaseError", CodeDatabaseError, 5000, 5999},
		{"CodeExternalError", CodeExternalError, 5000, 5999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code < tt.min || tt.code > tt.max {
				t.Errorf("%s = %d, expected to be in range [%d, %d]", tt.name, tt.code, tt.min, tt.max)
			}
		})
	}
}
