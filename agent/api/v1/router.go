package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

// TaskExecutor handles task execution
type TaskExecutor struct {
	// In-memory map for idempotency (requestId -> result)
	resultCache sync.Map
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor() *TaskExecutor {
	return &TaskExecutor{}
}

// ExecuteTaskRequest represents a task execution request
type ExecuteTaskRequest struct {
	RequestID string      `json:"requestId" binding:"required"`
	Type      string      `json:"type" binding:"required"`
	Payload   interface{} `json:"payload"`
}

// ExecuteTaskResponse represents a task execution response
type ExecuteTaskResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RequestID string `json:"requestId"`
		Status    string `json:"status"`
		Message   string `json:"message"`
	} `json:"data"`
}

// Execute handles POST /agent/v1/tasks/execute
func (e *TaskExecutor) Execute(c *gin.Context) {
	// No Bearer token check - mTLS handles authentication

	// Parse request body
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    2001,
			"message": "invalid request body",
			"data":    nil,
		})
		return
	}

	// Check idempotency: if requestId already processed, return cached result
	if cachedResult, ok := e.resultCache.Load(req.RequestID); ok {
		log.Printf("[IDEMPOTENT] Request %s already processed, returning cached result", req.RequestID)
		c.JSON(200, cachedResult)
		return
	}

	// Execute task based on type
	var message string
	var err error

	switch req.Type {
	case "apply_config":
		message, err = e.executeApplyConfig(req.RequestID, req.Payload)
	case "reload":
		message, err = e.executeReload(req.RequestID, req.Payload)
	case "purge_cache":
		message, err = e.executePurgeCache(req.RequestID, req.Payload)
	default:
		c.JSON(400, gin.H{
			"code":    2002,
			"message": fmt.Sprintf("unknown task type: %s", req.Type),
			"data":    nil,
		})
		return
	}

	// Build response
	resp := ExecuteTaskResponse{
		Code:    0,
		Message: "success",
	}
	resp.Data.RequestID = req.RequestID

	if err != nil {
		resp.Data.Status = "failed"
		resp.Data.Message = err.Error()
		log.Printf("[ERROR] Task %s (%s) failed: %v", req.RequestID, req.Type, err)
	} else {
		resp.Data.Status = "success"
		resp.Data.Message = message
		log.Printf("[SUCCESS] Task %s (%s) completed: %s", req.RequestID, req.Type, message)
	}

	// Cache result for idempotency
	e.resultCache.Store(req.RequestID, resp)

	// Return response
	c.JSON(200, resp)
}

// executeApplyConfig simulates apply_config task
func (e *TaskExecutor) executeApplyConfig(requestID string, payload interface{}) (string, error) {
	// Write payload to file
	filename := fmt.Sprintf("/tmp/cmdb_apply_config_%s.json", requestID)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Config applied and saved to %s", filename), nil
}

// executeReload simulates reload task
func (e *TaskExecutor) executeReload(requestID string, payload interface{}) (string, error) {
	// Write to file
	filename := fmt.Sprintf("/tmp/cmdb_reload_%s.txt", requestID)
	content := fmt.Sprintf("Reload task executed at %s\n", requestID)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Reload completed and logged to %s", filename), nil
}

// executePurgeCache simulates purge_cache task
func (e *TaskExecutor) executePurgeCache(requestID string, payload interface{}) (string, error) {
	// Write payload to file
	filename := fmt.Sprintf("/tmp/cmdb_purge_cache_%s.json", requestID)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Cache purged and saved to %s", filename), nil
}

// SetupRouter sets up the agent API v1 routes
func SetupRouter(r *gin.Engine) {
	executor := NewTaskExecutor()

	v1 := r.Group("/agent/v1")
	{
		tasks := v1.Group("/tasks")
		{
			tasks.POST("/execute", executor.Execute)
		}
	}
}
