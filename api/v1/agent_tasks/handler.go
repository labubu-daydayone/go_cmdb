package agent_tasks

import (
	"go_cmdb/internal/httpx"
	"encoding/json"
	"log"
	"strconv"

	"go_cmdb/internal/agent"
	"go_cmdb/internal/config"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler handles agent task related requests
type Handler struct {
	db         *gorm.DB
	dispatcher *agent.Dispatcher
}

// NewHandler creates a new agent task handler
func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	dispatcher, err := agent.NewDispatcher(db, cfg)
	if err != nil {
		log.Printf("⚠ Failed to create dispatcher: %v", err)
	}

	return &Handler{
		db:         db,
		dispatcher: dispatcher,
	}
}

// List handles GET /api/v1/agent-tasks
func (h *Handler) List(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	nodeIDStr := c.Query("nodeId")
	taskType := c.Query("type")
	status := c.Query("status")
	releaseTaskIDStr := c.Query("releaseTaskId")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Build query
	query := h.db.Model(&model.AgentTask{})

	// Filter by nodeId
	if nodeIDStr != "" {
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err == nil && nodeID > 0 {
			query = query.Where("node_id = ?", nodeID)
		}
	}

	// Filter by type
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}

	// Filter by status
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by releaseTaskId (via payload JSON)
	if releaseTaskIDStr != "" {
		releaseTaskID, err := strconv.ParseInt(releaseTaskIDStr, 10, 64)
		if err == nil && releaseTaskID > 0 {
			// 使用 JSON_EXTRACT 或 LIKE 查询
			query = query.Where("payload LIKE ?", "%\\\"releaseTaskId\\\":"+releaseTaskIDStr+"%")
		}
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count tasks", err))
		return
	}

	// Query tasks
	var tasks []model.AgentTask
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query tasks", err))
		return
	}

	// Return response
	httpx.OK(c, gin.H{
		"items":     tasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetByID handles GET /api/v1/agent-tasks/:id
func (h *Handler) GetByID(c *gin.Context) {
	// Parse id
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid task id"))
		return
	}

	// Query task
	var task model.AgentTask
	if err := h.db.First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("task not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query task", err))
		return
	}

	// Return response
	httpx.OK(c, task)
}

// Create handles POST /api/v1/agent-tasks/create
func (h *Handler) Create(c *gin.Context) {
	// Parse request body
	var req struct {
		NodeID  int             `json:"nodeId" binding:"required"`
		Type    string          `json:"type" binding:"required"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// Validate type
	if req.Type != model.TaskTypePurgeCache && req.Type != model.TaskTypeApplyConfig && req.Type != model.TaskTypeReload {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid task type"))
		return
	}

	// Check if node exists
	var node model.Node
	if err := h.db.First(&node, req.NodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query node", err))
		return
	}

	// Generate request ID
	requestID := uuid.New().String()

	// Create task
	task := model.AgentTask{
		NodeID:    uint(req.NodeID),
		Type:      req.Type,
		Payload:   string(req.Payload),
		Status:    model.TaskStatusPending,
		Attempts:  0,
		RequestID: requestID,
	}

	if err := h.db.Create(&task).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create task", err))
		return
	}

	// Dispatch task immediately (synchronous)
	go func() {
		if err := h.dispatcher.DispatchTask(&task); err != nil {
			// Error already logged in dispatcher
		}
	}()

	// Return response
	httpx.OK(c, task)
}

// Retry handles POST /api/v1/agent-tasks/retry
func (h *Handler) Retry(c *gin.Context) {
	// Parse request body
	var req struct {
		ID int `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// Query task
	var task model.AgentTask
	if err := h.db.First(&task, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("task not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query task", err))
		return
	}

	// Check if task is failed
	if task.Status != model.TaskStatusFailed {
		httpx.FailErr(c, httpx.ErrParamInvalid("only failed tasks can be retried"))
		return
	}

	// Reset task status to pending
	task.Status = model.TaskStatusPending
	task.LastError = ""
	if err := h.db.Save(&task).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update task", err))
		return
	}

	// Dispatch task immediately (synchronous)
	go func() {
		if err := h.dispatcher.DispatchTask(&task); err != nil {
			// Error already logged in dispatcher
		}
	}()

	// Return response
	httpx.OK(c, task)
}
