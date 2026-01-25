package config

import (
	"go_cmdb/internal/httpx"
	"encoding/json"
	"log"
	"strconv"

	"go_cmdb/internal/agent"
	"go_cmdb/internal/config"
	"go_cmdb/internal/configgen"
	"go_cmdb/internal/configver"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler handles configuration management requests
type Handler struct {
	db         *gorm.DB
	configVer  *configver.Service
	aggregator *configgen.Aggregator
	dispatcher *agent.Dispatcher
}

// NewHandler creates a new configuration handler
func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	dispatcher, err := agent.NewDispatcher(db, cfg)
	if err != nil {
		log.Printf("⚠ Failed to create dispatcher: %v", err)
	}

	return &Handler{
		db:         db,
		configVer:  configver.NewService(db),
		aggregator: configgen.NewAggregator(db),
		dispatcher: dispatcher,
	}
}

// ApplyRequest represents the request for applying configuration
type ApplyRequest struct {
	NodeID int    `json:"nodeId" binding:"required"`
	Reason string `json:"reason"`
}

// ApplyResponse represents the response for applying configuration
type ApplyResponse struct {
	Version int64 `json:"version"`
	TaskID  int   `json:"taskId"`
}

// Apply handles POST /api/v1/config/apply
func (h *Handler) Apply(c *gin.Context) {
	var req ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Check if node exists
	var node model.Node
	if err := h.db.First(&node, req.NodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("query failed", err))
		return
	}

	// Step 1: Create config version with placeholder payload to get database-generated version ID
	// This ensures version is globally unique and incrementing
	configVersion, err := h.configVer.CreateVersion(req.NodeID, "{}", req.Reason)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to create config version", err))
		return
	}

	// Step 2: Generate payload using the database-generated version
	payload, err := h.aggregator.GeneratePayload(req.NodeID, configVersion.Version)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to generate payload", err))
		return
	}

	// Serialize payload
	payloadJSON, err := h.aggregator.SerializePayload(payload)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to serialize payload", err))
		return
	}

	// Step 3: Update config version with actual payload
	if err := h.db.Model(configVersion).Update("payload", payloadJSON).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update payload", err))
		return
	}

	// Step 4: Create agent task
	requestID := uuid.New().String()
	task := &model.AgentTask{
		RequestID: requestID,
		NodeID:    req.NodeID,
		Type:      model.TaskTypeApplyConfig,
		Payload:   payloadJSON,
		Status:    model.TaskStatusPending,
		Attempts:  0,
	}

	if err := h.db.Create(task).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create task", err))
		return
	}

	// Step 5: Dispatch task to agent (async)
	go func() {
		if err := h.dispatcher.DispatchTask(task); err != nil {
			log.Printf("Failed to dispatch task %d: %v", task.ID, err)
		}
	}()

	// Return response
	httpx.OK(c, ApplyResponse{
		Version: configVersion.Version,
		TaskID:  task.ID,
	})
}

// ListVersionsRequest represents the request for listing versions
type ListVersionsRequest struct {
	NodeID   *int `form:"nodeId"`
	Page     int  `form:"page"`
	PageSize int  `form:"pageSize"`
}

// ListVersions handles GET /api/v1/config/versions
func (h *Handler) ListVersions(c *gin.Context) {
	var req ListVersionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Query versions
	versions, total, err := h.configVer.ListVersions(req.NodeID, req.Page, req.PageSize)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("query failed", err))
		return
	}

	httpx.OK(c, gin.H{
		"items":     versions,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

// GetVersionRequest represents the request for getting a specific version
type GetVersionRequest struct {
	Version string `uri:"version" binding:"required"`
}

// GetVersion handles GET /api/v1/config/versions/:version
func (h *Handler) GetVersion(c *gin.Context) {
	var req GetVersionRequest
	if err := c.ShouldBindUri(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Parse version
	version, err := strconv.ParseInt(req.Version, 10, 64)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid version format"))
		return
	}

	// Query version
	configVersion, err := h.configVer.GetByVersion(version)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("query failed", err))
		return
	}

	if configVersion == nil {
		httpx.FailErr(c, httpx.ErrNotFound("version not found"))
		return
	}

	// Parse payload for better display
	var payload interface{}
	if err := json.Unmarshal([]byte(configVersion.Payload), &payload); err != nil {
		// If parse fails, return raw payload
		httpx.OK(c, configVersion)
		return
	}

	// Return with parsed payload
	httpx.OK(c, gin.H{
		"id":        configVersion.ID,
		"version":   configVersion.Version,
		"nodeId":    configVersion.NodeID,
		"payload":   payload,
		"status":    configVersion.Status,
		"appliedAt": configVersion.AppliedAt,
		"createdAt": configVersion.CreatedAt,
	})
}
