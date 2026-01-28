package node_ips

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/nodeip"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles node IP related requests
type Handler struct {
	service *nodeip.Service
}

// NewHandler creates a new node IP handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		service: nodeip.NewService(db),
	}
}

// List handles GET /api/v1/node-ips
func (h *Handler) List(c *gin.Context) {
	// Optional nodeId filter
	var nodeID *int
	if nodeIDStr := c.Query("nodeId"); nodeIDStr != "" {
		id, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid("invalid nodeId"))
			return
		}
		nodeID = &id
	}

	items, err := h.service.ListNodeIPs(nodeID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to list node IPs", err))
		return
	}

	httpx.OK(c, gin.H{
		"items": items,
	})
}

// DisableRequest represents the disable request
type DisableRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Disable handles POST /api/v1/node-ips/disable
func (h *Handler) Disable(c *gin.Context) {
	var req DisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Check if all IDs exist
	existingIDs, err := h.service.CheckIPsExist(req.IDs)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check IPs", err))
		return
	}

	if len(existingIDs) != len(req.IDs) {
		httpx.FailErr(c, httpx.ErrNotFound("some IPs not found"))
		return
	}

	// Disable IPs (idempotent)
	if err := h.service.DisableNodeIPs(req.IDs); err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to disable node IPs", err))
		return
	}

	httpx.OK(c, nil)
}

// EnableRequest represents the enable request
type EnableRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Enable handles POST /api/v1/node-ips/enable
func (h *Handler) Enable(c *gin.Context) {
	var req EnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Check if all IDs exist
	existingIDs, err := h.service.CheckIPsExist(req.IDs)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check IPs", err))
		return
	}

	if len(existingIDs) != len(req.IDs) {
		httpx.FailErr(c, httpx.ErrNotFound("some IPs not found"))
		return
	}

	// Enable IPs (idempotent)
	if err := h.service.EnableNodeIPs(req.IDs); err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to enable node IPs", err))
		return
	}

	httpx.OK(c, nil)
}
