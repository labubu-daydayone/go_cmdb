package agent_exec

import (
	"encoding/json"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles agent execution requests
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new agent execution handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// extractNodeID extracts nodeId from mTLS client cert or X-Node-Id header
func (h *Handler) extractNodeID(c *gin.Context) (int64, error) {
	// TODO: Priority 1 - mTLS client cert parsing (if framework supports)
	// if nodeID, ok := extractFromClientCert(c); ok {
	//     return nodeID, nil
	// }

	// Priority 2 - X-Node-Id header (for testing)
	nodeIDStr := c.GetHeader("X-Node-Id")
	if nodeIDStr == "" {
		return 0, httpx.ErrParamInvalid("missing X-Node-Id header")
	}

	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil || nodeID <= 0 {
		return 0, httpx.ErrParamInvalid("invalid X-Node-Id")
	}

	return nodeID, nil
}

// mapStatusToAPI maps database status to API status
func mapStatusToAPI(dbStatus string) string {
	if dbStatus == "success" {
		return "succeeded"
	}
	return dbStatus
}

// mapStatusToDB maps API status to database status
func mapStatusToDB(apiStatus string) string {
	if apiStatus == "succeeded" {
		return "success"
	}
	return apiStatus
}

// Pull handles GET /api/v1/agent/tasks/pull
func (h *Handler) Pull(c *gin.Context) {
	// Extract nodeId
	nodeID, err := h.extractNodeID(c)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		}
		return
	}

	// Parse limit
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Step 0: 补派发离线期间的 pending release_tasks
	dispatcher := service.NewAgentTaskDispatcher(h.db)
	dispatchPendingResult, err := dispatcher.EnsureDispatchPendingForNode(nodeID)
	if err != nil {
		log.Printf("[Agent Pull] Failed to dispatch pending tasks for node %d: %v", nodeID, err)
		// 不阻断 pull，继续返回已有任务
	}

	// Step 1: SELECT可领取任务id列表
	var taskIDs []int64
	err = h.db.Model(&model.AgentTask{}).
		Where("node_id = ?", nodeID).
		Where("status IN ?", []string{"pending", "retrying"}).
		Order("id ASC").
		Limit(limit).
		Pluck("id", &taskIDs).Error

	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query tasks", err))
		return
	}

	// No tasks to pull
	if len(taskIDs) == 0 {
		httpx.OK(c, gin.H{"items": []model.AgentTask{}})
		return
	}

	// Step 2: UPDATE领取任务（原子更新）
	result := h.db.Model(&model.AgentTask{}).
		Where("id IN ?", taskIDs).
		Where("status IN ?", []string{"pending", "retrying"}).
		Updates(map[string]interface{}{
			"status":     "running",
			"updated_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update task status", result.Error))
		return
	}

	// If no rows updated, return empty
	if result.RowsAffected == 0 {
		httpx.OK(c, gin.H{"items": []model.AgentTask{}})
		return
	}

	// Step 3: SELECT领取成功的任务
	var tasks []model.AgentTask
	err = h.db.Where("id IN ?", taskIDs).
		Where("status = ?", "running").
		Find(&tasks).Error

	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query updated tasks", err))
		return
	}

	// Map status to API format and convert payload to object
	type AgentTaskResponse struct {
		ID          int64                  `json:"id"`
		NodeID      int                    `json:"nodeId"`
		Type        string                 `json:"type"`
		Payload     map[string]interface{} `json:"payload"`
		Status      string                 `json:"status"`
		LastError   string                 `json:"lastError,omitempty"`
		CreatedAt   time.Time              `json:"createdAt"`
		UpdatedAt   time.Time              `json:"updatedAt"`
	}

	var responseTasks []AgentTaskResponse
	for i := range tasks {
		// Parse payload string to object
		var payloadObj map[string]interface{}
		if err := json.Unmarshal([]byte(tasks[i].Payload), &payloadObj); err != nil {
			// Skip tasks with invalid payload and log error
			log.Printf("[Agent Pull] Skip task %d due to invalid payload: %v", tasks[i].ID, err)
			continue
		}

		responseTasks = append(responseTasks, AgentTaskResponse{
			ID:        int64(tasks[i].ID),
			NodeID:    int(tasks[i].NodeID),
			Type:      tasks[i].Type,
			Payload:   payloadObj,
			Status:    mapStatusToAPI(tasks[i].Status),
			LastError: tasks[i].LastError,
			CreatedAt: tasks[i].CreatedAt,
			UpdatedAt: tasks[i].UpdatedAt,
		})
	}

	// 添加补派发统计字段到响应
	response := gin.H{"items": responseTasks}
	if dispatchPendingResult != nil {
		response["dispatchedBeforePull"] = dispatchPendingResult.DispatchedBeforePull
		response["dispatchedCreatedCount"] = dispatchPendingResult.DispatchedCreatedCount
		response["dispatchedSkippedCount"] = dispatchPendingResult.DispatchedSkippedCount
		response["pendingReleaseTaskTouchedCount"] = dispatchPendingResult.PendingReleaseTaskTouchedCount
	}
	httpx.OK(c, response)
}

// UpdateStatus handles POST /api/v1/agent/tasks/update-status
func (h *Handler) UpdateStatus(c *gin.Context) {
	// Extract nodeId
	nodeID, err := h.extractNodeID(c)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		}
		return
	}

	// Parse request body
	var req struct {
		ID        int64  `json:"id" binding:"required"`
		Status    string `json:"status" binding:"required"`
		LastError string `json:"lastError"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// Validate status
	if req.Status != "succeeded" && req.Status != "failed" {
		httpx.FailErr(c, httpx.ErrParamInvalid("status must be succeeded or failed"))
		return
	}

	// Validate lastError (optional for failed, must be empty for succeeded)
	if req.Status == "failed" {
		if len(req.LastError) > 2048 {
			httpx.FailErr(c, httpx.ErrParamInvalid("lastError too long (max 2048)"))
			return
		}
	}

	// Call the service layer to handle the update and propagate to release_task
	if err := service.UpdateTaskStatus(uint(nodeID), uint(req.ID), req.Status, req.LastError); err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update task status", err))
		return
	}

	httpx.OK(c, nil)
}
