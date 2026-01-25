package agent_identities

import (
	"go_cmdb/internal/httpx"
	"time"

	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles agent identity management
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new agent identity handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListRequest represents list agent identities request
type ListRequest struct {
	NodeID *int    `form:"nodeId"`
	Status *string `form:"status"`
}

// CreateRequest represents create agent identity request
type CreateRequest struct {
	NodeID          int    `json:"nodeId" binding:"required"`
	CertFingerprint string `json:"certFingerprint" binding:"required"`
}

// RevokeRequest represents revoke agent identity request
type RevokeRequest struct {
	NodeID int `json:"nodeId" binding:"required"`
}

// List handles GET /api/v1/agent-identities
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Build query
	query := h.db.Model(&model.AgentIdentity{})

	if req.NodeID != nil {
		query = query.Where("node_id = ?", *req.NodeID)
	}

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// Execute query
	var identities []model.AgentIdentity
	if err := query.Order("id DESC").Find(&identities).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("query failed", err))
		return
	}

	httpx.OK(c, gin.H{
		"items":  identities,
		"total": len(identities),
	})
}

// Create handles POST /api/v1/agent-identities/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
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

	// Check if identity already exists for this node
	var existing model.AgentIdentity
	if err := h.db.Where("node_id = ?", req.NodeID).First(&existing).Error; err == nil {
		httpx.FailErr(c, httpx.ErrAlreadyExists("agent identity already exists for this node"))
		return
	}

	// Check if fingerprint already exists
	if err := h.db.Where("cert_fingerprint = ?", req.CertFingerprint).First(&existing).Error; err == nil {
		httpx.FailErr(c, httpx.ErrAlreadyExists("certificate fingerprint already exists"))
		return
	}

	// Create identity
	identity := model.AgentIdentity{
		NodeID:          req.NodeID,
		CertFingerprint: req.CertFingerprint,
		Status:          model.AgentIdentityStatusActive,
		IssuedAt:        time.Now(),
	}

	if err := h.db.Create(&identity).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("create failed", err))
		return
	}

	httpx.OK(c, identity)
}

// Revoke handles POST /api/v1/agent-identities/revoke
func (h *Handler) Revoke(c *gin.Context) {
	var req RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Find identity
	var identity model.AgentIdentity
	if err := h.db.Where("node_id = ?", req.NodeID).First(&identity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("agent identity not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("query failed", err))
		return
	}

	// Check if already revoked
	if identity.Status == model.AgentIdentityStatusRevoked {
		httpx.FailErr(c, httpx.ErrAlreadyExists("agent identity already revoked"))
		return
	}

	// Revoke identity
	now := time.Now()
	identity.Status = model.AgentIdentityStatusRevoked
	identity.RevokedAt = &now

	if err := h.db.Save(&identity).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("update failed", err))
		return
	}

	httpx.OK(c, identity)
}
