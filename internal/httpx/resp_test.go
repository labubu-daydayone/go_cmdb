package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestOK(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		OK(c, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != CodeSuccess {
		t.Errorf("Expected code %d, got %d", CodeSuccess, resp.Code)
	}

	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Message)
	}

	if resp.Data == nil {
		t.Error("Expected data to be non-nil")
	}
}

func TestOKMsg(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		OKMsg(c, "custom message", gin.H{"result": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeSuccess {
		t.Errorf("Expected code %d, got %d", CodeSuccess, resp.Code)
	}

	if resp.Message != "custom message" {
		t.Errorf("Expected message 'custom message', got '%s'", resp.Message)
	}
}

func TestFail(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		Fail(c, http.StatusBadRequest, CodeParamMissing, "param missing")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeParamMissing {
		t.Errorf("Expected code %d, got %d", CodeParamMissing, resp.Code)
	}

	if resp.Message != "param missing" {
		t.Errorf("Expected message 'param missing', got '%s'", resp.Message)
	}

	if resp.Data != nil {
		t.Error("Expected data to be nil for error response")
	}
}

func TestFailErr(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		FailErr(c, ErrNotFound("resource not found"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeNotFound {
		t.Errorf("Expected code %d, got %d", CodeNotFound, resp.Code)
	}

	if resp.Message != "resource not found" {
		t.Errorf("Expected message 'resource not found', got '%s'", resp.Message)
	}

	if resp.Data != nil {
		t.Error("Expected data to be nil for error response")
	}
}

func TestFailErr_WithInternalError(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		// Internal error should be logged but not returned to client
		FailErr(c, ErrInternalError("internal error", nil))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeInternalError {
		t.Errorf("Expected code %d, got %d", CodeInternalError, resp.Code)
	}

	// Message should not contain internal error details
	if resp.Message != "internal error" {
		t.Errorf("Expected message 'internal error', got '%s'", resp.Message)
	}
}
