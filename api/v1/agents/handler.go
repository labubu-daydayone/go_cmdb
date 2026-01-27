package agents

import (
	"time"

	"go_cmdb/internal/bootstrap"
	"go_cmdb/internal/httpx"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles agent-related requests
type Handler struct {
	db         *gorm.DB
	tokenStore *bootstrap.TokenStore
	controlURL string
}

// NewHandler creates a new agents handler
func NewHandler(db *gorm.DB, tokenStore *bootstrap.TokenStore, controlURL string) *Handler {
	return &Handler{
		db:         db,
		tokenStore: tokenStore,
		controlURL: controlURL,
	}
}

// CreateBootstrapTokenRequest represents the request to create a bootstrap token
type CreateBootstrapTokenRequest struct {
	NodeID int `json:"nodeId" binding:"required"`
	TTLSec int `json:"ttlSec" binding:"required,min=60,max=86400"` // 1 minute to 24 hours
}

// CreateBootstrapToken creates a new bootstrap token for agent installation
// POST /api/v1/agents/bootstrap/token/create
func (h *Handler) CreateBootstrapToken(c *gin.Context) {
	var req CreateBootstrapTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Validate node exists
	var nodeExists bool
	if err := h.db.Table("nodes").Select("1").Where("id = ?", req.NodeID).Scan(&nodeExists).Error; err != nil {
		c.JSON(500, gin.H{"code": 5000, "message": "failed to validate node", "data": nil})
		return
	}
	if !nodeExists {
		httpx.FailErr(c, httpx.ErrParamInvalid("node not found"))
		return
	}

	// Create token in Redis
	token, err := h.tokenStore.CreateToken(c.Request.Context(), req.NodeID, req.TTLSec)
	if err != nil {
		c.JSON(500, gin.H{"code": 5000, "message": "failed to create token: " + err.Error(), "data": nil})
		return
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(req.TTLSec) * time.Second)

	// Build install URL
	installURL := h.controlURL + "/bootstrap/agent/install.sh?token=" + token

	httpx.OK(c, gin.H{
		"token":      token,
		"installUrl": installURL,
		"expiresAt":  expiresAt.Format("2006-01-02T15:04:05-07:00"),
	})
}
