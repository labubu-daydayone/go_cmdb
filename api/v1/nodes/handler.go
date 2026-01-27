package nodes

import (
	"fmt"
	"strings"

	"go_cmdb/internal/dto"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/nodes"
	"go_cmdb/internal/pki"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRequest represents list nodes request
type ListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Name     string `form:"name"`
	IP       string `form:"ip"`
	Status   string `form:"status"`
	Enabled  *bool  `form:"enabled"`
}

// ListResponse represents list nodes response
type ListResponse struct {
	Items    []dto.NodeDTO `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// CreateRequest represents create node request
type CreateRequest struct {
	Name      string          `json:"name" binding:"required"`
	MainIP    string          `json:"mainIp" binding:"required"`
	AgentPort int             `json:"agentPort"`
	Enabled   *bool           `json:"enabled"`
	SubIPs    []SubIPRequest  `json:"subIps"`
}

// SubIPRequest represents sub IP in create/update request
type SubIPRequest struct {
	ID      int    `json:"id"`
	IP      string `json:"ip" binding:"required"`
	Enabled *bool  `json:"enabled"`
}

// UpdateRequest represents update node request
type UpdateRequest struct {
	ID        int             `json:"id" binding:"required"`
	Name      *string         `json:"name"`
	MainIP    *string         `json:"mainIp"`
	AgentPort *int            `json:"agentPort"`
	Enabled   *bool           `json:"enabled"`
	Status    *string         `json:"status"`
	SubIPs    []SubIPRequest  `json:"subIps"`
}

// DeleteRequest represents delete nodes request
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// AddSubIPsRequest represents add sub IPs request
type AddSubIPsRequest struct {
	NodeID int            `json:"nodeId" binding:"required"`
	SubIPs []SubIPRequest `json:"subIps" binding:"required,min=1"`
}

// DeleteSubIPsRequest represents delete sub IPs request
type DeleteSubIPsRequest struct {
	NodeID   int   `json:"nodeId" binding:"required"`
	SubIPIDs []int `json:"subIpIds" binding:"required,min=1"`
}

// ToggleSubIPRequest represents toggle sub IP enabled request
type ToggleSubIPRequest struct {
	NodeID  int  `json:"nodeId" binding:"required"`
	SubIPID int  `json:"subIpId" binding:"required"`
	Enabled bool `json:"enabled"`
}

// Handler handles nodes API
type Handler struct {
	db              *gorm.DB
	identityService *nodes.IdentityService
}

// NewHandler creates a new nodes handler
func NewHandler(db *gorm.DB, caManager *pki.CAManager) *Handler {
	return &Handler{
		db:              db,
		identityService: nodes.NewIdentityService(db, caManager),
	}
}

// List handles GET /api/v1/nodes
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 15
	}

	// Build query
	query := h.db.Model(&model.Node{})

	// Name filter (fuzzy)
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// IP filter (main_ip or sub_ip)
	if req.IP != "" {
		subQuery := h.db.Model(&model.NodeSubIP{}).
			Select("node_id").
			Where("ip LIKE ?", "%"+req.IP+"%")
		
		query = query.Where("main_ip LIKE ? OR id IN (?)", "%"+req.IP+"%", subQuery)
	}

	// Status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Enabled filter
	if req.Enabled != nil {
		query = query.Where("enabled = ?", *req.Enabled)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count nodes", err))
		return
	}

	// Fetch nodes with pagination
	var nodes []model.Node
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("id DESC").
		Find(&nodes).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch nodes", err))
		return
	}

	// Convert to DTO
	items := make([]dto.NodeDTO, len(nodes))
	for i, node := range nodes {
		items[i] = dto.NodeDTO{
			ID:        node.ID,
			Name:      node.Name,
			MainIp:    node.MainIP,
			AgentPort: node.AgentPort,
			Enabled:   node.Enabled,
			Status:    string(node.Status),
			CreatedAt: node.CreatedAt,
			UpdatedAt: node.UpdatedAt,
		}
	}

	httpx.OK(c, ListResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create handles POST /api/v1/nodes/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Validate mainIP not empty
	if strings.TrimSpace(req.MainIP) == "" {
		httpx.FailErr(c, httpx.ErrParamInvalid("mainIp cannot be empty"))
		return
	}

	// Check name uniqueness
	var count int64
	if err := h.db.Model(&model.Node{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if count > 0 {
		httpx.FailErr(c, httpx.ErrAlreadyExists("node name already exists"))
		return
	}

	// Check subIPs uniqueness within request
	ipMap := make(map[string]bool)
	for _, subIP := range req.SubIPs {
		if ipMap[subIP.IP] {
			httpx.FailErr(c, httpx.ErrParamInvalid(fmt.Sprintf("duplicate sub IP: %s", subIP.IP)))
			return
		}
		ipMap[subIP.IP] = true
	}

	// Create node
	node := model.Node{
		Name:      req.Name,
		MainIP:    req.MainIP,
		AgentPort: req.AgentPort,
		Enabled:   true,
		Status:    model.NodeStatusOffline,
	}

	if req.Enabled != nil {
		node.Enabled = *req.Enabled
	}
	if node.AgentPort == 0 {
		node.AgentPort = 8080
	}

	// Create sub IPs
	for _, subIP := range req.SubIPs {
		enabled := true
		if subIP.Enabled != nil {
			enabled = *subIP.Enabled
		}
		node.SubIPs = append(node.SubIPs, model.NodeSubIP{
			IP:      subIP.IP,
			Enabled: enabled,
		})
	}

	// Use transaction to ensure atomicity of node creation and identity generation
	var identity *model.AgentIdentity
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Create node
		if err := tx.Create(&node).Error; err != nil {
			return fmt.Errorf("failed to create node: %w", err)
		}

		// Generate identity
		var err error
		identity, err = h.identityService.CreateIdentity(tx, node.ID, node.Name)
		if err != nil {
			return fmt.Errorf("failed to create identity: %w", err)
		}

		return nil
	})

	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create node with identity", err))
		return
	}

	// Build DTO response
	response := dto.NodeWithIdentityDTO{
		ID:        node.ID,
		Name:      node.Name,
		MainIp:    node.MainIP,
		AgentPort: node.AgentPort,
		Enabled:   node.Enabled,
		Status:    string(node.Status),
		Identity: &dto.IdentityDTO{
			ID:          identity.ID,
			Fingerprint: identity.Fingerprint,
		},
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}

	httpx.OK(c, response)
}

// Update handles POST /api/v1/nodes/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Check node exists
	var node model.Node
	if err := h.db.First(&node, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node", err))
		return
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Name != nil {
		// Check name uniqueness (exclude current node)
		var count int64
		if err := h.db.Model(&model.Node{}).
			Where("name = ? AND id != ?", *req.Name, req.ID).
			Count(&count).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
			return
		}
		if count > 0 {
			httpx.FailErr(c, httpx.ErrAlreadyExists("node name already exists"))
			return
		}
		updates["name"] = *req.Name
	}

	if req.MainIP != nil {
		updates["main_ip"] = *req.MainIP
	}

	if req.AgentPort != nil {
		updates["agent_port"] = *req.AgentPort
	}

	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// Update node
	if len(updates) > 0 {
		if err := h.db.Model(&node).Updates(updates).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to update node", err))
			return
		}
	}

	// Handle sub IPs update
	if req.SubIPs != nil {
		// Delete all existing sub IPs
		if err := h.db.Where("node_id = ?", req.ID).Delete(&model.NodeSubIP{}).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete old sub IPs", err))
			return
		}

		// Create new sub IPs
		for _, subIP := range req.SubIPs {
			enabled := true
			if subIP.Enabled != nil {
				enabled = *subIP.Enabled
			}
			newSubIP := model.NodeSubIP{
				NodeID:  req.ID,
				IP:      subIP.IP,
				Enabled: enabled,
			}
			if err := h.db.Create(&newSubIP).Error; err != nil {
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP", err))
				return
			}
		}
	}

	// Reload node
	if err := h.db.First(&node, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload node", err))
		return
	}

	// Build DTO response
	response := dto.NodeDTO{
		ID:        node.ID,
		Name:      node.Name,
		MainIp:    node.MainIP,
		AgentPort: node.AgentPort,
		Enabled:   node.Enabled,
		Status:    string(node.Status),
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}

	httpx.OK(c, response)
}

// Get handles GET /api/v1/nodes/:id
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.FailErr(c, httpx.ErrParamMissing("id is required"))
		return
	}

	var node model.Node
	if err := h.db.Preload("SubIPs").First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node", err))
		return
	}

	// Get identity
	var identity model.AgentIdentity
	if err := h.db.Where("node_id = ? AND status = ?", node.ID, "active").First(&identity).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find identity", err))
			return
		}
		// Identity not found is not an error, just return node without identity
	}

	// Convert sub IPs to DTO
	subIps := make([]dto.SubIpDTO, len(node.SubIPs))
	for i, subIP := range node.SubIPs {
		subIps[i] = dto.SubIpDTO{
			ID:      subIP.ID,
			IP:      subIP.IP,
			Enabled: subIP.Enabled,
		}
	}

	// Build DTO response
	response := dto.NodeDetailDTO{
		ID:        node.ID,
		Name:      node.Name,
		MainIp:    node.MainIP,
		AgentPort: node.AgentPort,
		Enabled:   node.Enabled,
		Status:    string(node.Status),
		SubIps:    subIps,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}

	if identity.ID != 0 {
		response.Identity = &dto.IdentityDTO{
			ID:          identity.ID,
			Fingerprint: identity.Fingerprint,
		}
	}

	httpx.OK(c, response)
}

// Delete handles POST /api/v1/nodes/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Delete nodes (cascade delete will handle sub IPs)
	result := h.db.Delete(&model.Node{}, req.IDs)
	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete nodes", result.Error))
		return
	}

	httpx.OK(c, gin.H{
		"affected": result.RowsAffected,
	})
}

// AddSubIPs handles POST /api/v1/nodes/sub-ips/add
func (h *Handler) AddSubIPs(c *gin.Context) {
	var req AddSubIPsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Check node exists
	var node model.Node
	if err := h.db.First(&node, req.NodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node", err))
		return
	}

	// Check subIPs uniqueness within request
	ipMap := make(map[string]bool)
	for _, subIP := range req.SubIPs {
		if ipMap[subIP.IP] {
			httpx.FailErr(c, httpx.ErrParamInvalid(fmt.Sprintf("duplicate sub IP: %s", subIP.IP)))
			return
		}
		ipMap[subIP.IP] = true
	}

	// Create sub IPs
	for _, subIP := range req.SubIPs {
		enabled := true
		if subIP.Enabled != nil {
			enabled = *subIP.Enabled
		}
		newSubIP := model.NodeSubIP{
			NodeID:  req.NodeID,
			IP:      subIP.IP,
			Enabled: enabled,
		}
		if err := h.db.Create(&newSubIP).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP", err))
			return
		}
	}

	httpx.OK(c, gin.H{
		"message": "sub IPs added successfully",
	})
}

// DeleteSubIPs handles POST /api/v1/nodes/sub-ips/delete
func (h *Handler) DeleteSubIPs(c *gin.Context) {
	var req DeleteSubIPsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Delete sub IPs
	result := h.db.Where("node_id = ? AND id IN ?", req.NodeID, req.SubIPIDs).Delete(&model.NodeSubIP{})
	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete sub IPs", result.Error))
		return
	}

	httpx.OK(c, gin.H{
		"affected": result.RowsAffected,
	})
}

// ToggleSubIP handles POST /api/v1/nodes/sub-ips/toggle
func (h *Handler) ToggleSubIP(c *gin.Context) {
	var req ToggleSubIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Update sub IP enabled status
	result := h.db.Model(&model.NodeSubIP{}).
		Where("node_id = ? AND id = ?", req.NodeID, req.SubIPID).
		Update("enabled", req.Enabled)
	
	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to toggle sub IP", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		httpx.FailErr(c, httpx.ErrNotFound("sub IP not found"))
		return
	}

	httpx.OK(c, gin.H{
		"message": "sub IP toggled successfully",
	})
}

// GetIdentity handles GET /api/v1/nodes/:id/identity
func (h *Handler) GetIdentity(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		httpx.FailErr(c, httpx.ErrParamInvalid("node ID is required"))
		return
	}

	// Convert to int
	var id int
	if _, err := fmt.Sscanf(nodeID, "%d", &id); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid node ID"))
		return
	}

	// Check if node exists
	var node model.Node
	if err := h.db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node", err))
		return
	}

	// Get identity
	identity, err := h.identityService.GetIdentityByNodeID(id)
	if err != nil {
		httpx.FailErr(c, httpx.ErrNotFound("identity not found"))
		return
	}

	httpx.OK(c, identity)
}

// RevokeIdentity handles POST /api/v1/nodes/:id/identity/revoke
func (h *Handler) RevokeIdentity(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		httpx.FailErr(c, httpx.ErrParamInvalid("node ID is required"))
		return
	}

	// Convert to int
	var id int
	if _, err := fmt.Sscanf(nodeID, "%d", &id); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid node ID"))
		return
	}

	// Check if node exists
	var node model.Node
	if err := h.db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node", err))
		return
	}

	// Revoke identity
	if err := h.identityService.RevokeIdentity(id); err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to revoke identity", err))
		return
	}

	httpx.OK(c, gin.H{"message": "identity revoked"})
}
