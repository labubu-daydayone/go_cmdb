package node_groups

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRequest represents list node groups request
type ListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Name     string `form:"name"`
	DomainID *int   `form:"domainId"`
	Status   string `form:"status"`
}

// ListResponse represents list node groups response
type ListResponse struct {
	Items    []NodeGroupItem `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// NodeGroupItem represents a node group in list response
type NodeGroupItem struct {
	model.NodeGroup
	IPCount int `json:"ip_count"`
}

// CreateRequest represents create node group request
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IPIDs    []int  `json:"ipIds" binding:"required,min=1"`
}

// UpdateRequest represents update node group request
type UpdateRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	IPIDs    []int   `json:"ipIds"`
}

// DeleteRequest represents delete node groups request
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Handler handles node groups API
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new node groups handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// generateCNAMEPrefix generates a random CNAME prefix
func generateCNAMEPrefix() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "ng-" + hex.EncodeToString(bytes)
}

// createDNSRecordsForAllCDNDomains creates A records for all CDN domains × all available IPs
func (h *Handler) createDNSRecordsForAllCDNDomains(tx *gorm.DB, nodeGroup *model.NodeGroup, cdnDomains []model.Domain, ipIDs []int) error {
	if len(ipIDs) == 0 {
		return nil
	}

	// Fetch IPs and filter by enabled=true
	var ips []model.NodeIP
	if err := tx.Where("id IN ? AND enabled = ?", ipIDs, true).Find(&ips).Error; err != nil {
		return fmt.Errorf("failed to fetch IPs: %w", err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("no available IPs found")
	}

	// Create A records for each CDN domain × each available IP
	for _, domain := range cdnDomains {
		for _, ip := range ips {
			dnsRecord := model.DomainDNSRecord{
				DomainID:  domain.ID,
				Type:      model.DNSRecordTypeA,
				Name:      nodeGroup.CNAMEPrefix,
				Value:     ip.IP,
				OwnerType: "node_group",
				OwnerID:   nodeGroup.ID,
				Status:    model.DNSRecordStatusPending,
			}

			if err := tx.Create(&dnsRecord).Error; err != nil {
				return fmt.Errorf("failed to create DNS record for domain %s, IP %s: %w", domain.Domain, ip.IP, err)
			}
		}
	}

	return nil
}

// createDNSRecordsForNodeGroup creates A records for node group sub IPs (deprecated, kept for compatibility)
func (h *Handler) createDNSRecordsForNodeGroup(tx *gorm.DB, nodeGroup *model.NodeGroup, ipIDs []int) error {
	if len(ipIDs) == 0 {
		return nil
	}

	// Fetch sub IPs
	var ips []model.NodeIP
	if err := tx.Where("id IN ?", ipIDs).Find(&ips).Error; err != nil {
		return fmt.Errorf("failed to fetch sub IPs: %w", err)
	}

	// Create A records for each sub IP
	for _, ip := range ips {
		dnsRecord := model.DomainDNSRecord{
			DomainID:  nodeGroup.DomainID,
			Type:      model.DNSRecordTypeA,
			Name:      nodeGroup.CNAMEPrefix,
			Value:     ip.IP,
			OwnerType: "node_group",
			OwnerID:   nodeGroup.ID,
			Status:    model.DNSRecordStatusPending,
		}

		if err := tx.Create(&dnsRecord).Error; err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}
	}

	return nil
}

// markDNSRecordsAsError marks DNS records as error for a node group
func (h *Handler) markDNSRecordsAsError(tx *gorm.DB, nodeGroupID int, reason string) error {
	updates := map[string]interface{}{
		"status":     model.DNSRecordStatusError,
		"last_error": reason,
	}

	if err := tx.Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id = ?", "node_group", nodeGroupID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark DNS records as error: %w", err)
	}

	return nil
}

// List handles GET /api/v1/node-groups
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
	query := h.db.Model(&model.NodeGroup{})

	// Name filter (fuzzy)
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// DomainID filter
	if req.DomainID != nil {
		query = query.Where("domain_id = ?", *req.DomainID)
	}

	// Status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count node groups", err))
		return
	}

	// Fetch node groups with pagination
	var nodeGroups []model.NodeGroup
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Preload("Domain").
		Offset(offset).
		Limit(req.PageSize).
		Order("id DESC").
		Find(&nodeGroups).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch node groups", err))
		return
	}

	// Build response with sub IP count
	items := make([]NodeGroupItem, len(nodeGroups))
	for i, ng := range nodeGroups {
		var count int64
		h.db.Model(&model.NodeGroupIP{}).Where("node_group_id = ?", ng.ID).Count(&count)
		
		items[i] = NodeGroupItem{
			NodeGroup:  ng,
			IPCount: int(count),
		}
	}

	httpx.OK(c, ListResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create handles POST /api/v1/node-groups/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Fetch all CDN domains (purpose=cdn, status=active)
	var cdnDomains []model.Domain
	if err := h.db.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch CDN domains", err))
		return
	}

	if len(cdnDomains) == 0 {
		httpx.FailErr(c, httpx.ErrNotFound("no active CDN domains found"))
		return
	}

	// Use first CDN domain as primary domain (for compatibility)
	primaryDomain := cdnDomains[0]

	// Check name uniqueness
	var count int64
	if err := h.db.Model(&model.NodeGroup{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if count > 0 {
		httpx.FailErr(c, httpx.ErrAlreadyExists("node group name already exists"))
		return
	}

	// Generate CNAME prefix
	var cnamePrefix string
	for {
		cnamePrefix = generateCNAMEPrefix()
		var existCount int64
		if err := h.db.Model(&model.NodeGroup{}).Where("cname_prefix = ?", cnamePrefix).Count(&existCount).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to check cname_prefix uniqueness", err))
			return
		}
		if existCount == 0 {
			break
		}
	}

		// Use primary domain for CNAME (for display)
		cname := cnamePrefix + "." + primaryDomain.Domain

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	nodeGroup := model.NodeGroup{
		Name:        req.Name,
		Description: req.Description,
		DomainID:    primaryDomain.ID,
		CNAMEPrefix: cnamePrefix,
		CNAME:       cname,
		Status:      model.NodeGroupStatusActive,
	}

	if err := tx.Create(&nodeGroup).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create node group", err))
		return
	}

	if len(req.IPIDs) > 0 {
		for _, ipID := range req.IPIDs {
			mapping := model.NodeGroupIP{
				NodeGroupID: nodeGroup.ID,
				IPID:     ipID,
			}
			if err := tx.Create(&mapping).Error; err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP mapping", err))
				return
			}
		}

		// Create DNS records for all CDN domains × all available IPs
		if err := h.createDNSRecordsForAllCDNDomains(tx, &nodeGroup, cdnDomains, req.IPIDs); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS records", err))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	if err := h.db.Preload("Domain").Preload("IPs").First(&nodeGroup, nodeGroup.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload node group", err))
		return
	}

	httpx.OK(c, nodeGroup)
}

// Update handles POST /api/v1/node-groups/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	var nodeGroup model.NodeGroup
	if err := h.db.First(&nodeGroup, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node group", err))
		return
	}

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updates := make(map[string]interface{})

	if req.Name != nil {
		var count int64
		if err := tx.Model(&model.NodeGroup{}).Where("name = ? AND id != ?", *req.Name, req.ID).Count(&count).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
			return
		}
		if count > 0 {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrAlreadyExists("node group name already exists"))
			return
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := tx.Model(&nodeGroup).Updates(updates).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to update node group", err))
			return
		}
	}

	if req.IPIDs != nil {
		if err := h.markDNSRecordsAsError(tx, req.ID, "sub IPs updated"); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark old DNS records as error", err))
			return
		}

		if err := tx.Where("node_group_id = ?", req.ID).Delete(&model.NodeGroupIP{}).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete old sub IP mappings", err))
			return
		}

		if len(req.IPIDs) > 0 {
			for _, ipID := range req.IPIDs {
				mapping := model.NodeGroupIP{
					NodeGroupID: req.ID,
					IPID:     ipID,
				}
				if err := tx.Create(&mapping).Error; err != nil {
					tx.Rollback()
					httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP mapping", err))
					return
				}
			}

			if err := h.createDNSRecordsForNodeGroup(tx, &nodeGroup, req.IPIDs); err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS records", err))
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	if err := h.db.Preload("Domain").Preload("IPs").First(&nodeGroup, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload node group", err))
		return
	}

	httpx.OK(c, nodeGroup)
}

// Delete handles POST /api/v1/node-groups/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, id := range req.IDs {
		if err := h.markDNSRecordsAsError(tx, id, "node group deleted"); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark DNS records as error", err))
			return
		}
	}

	result := tx.Delete(&model.NodeGroup{}, req.IDs)
	if result.Error != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete node groups", result.Error))
		return
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	httpx.OK(c, gin.H{
		"deletedCount": result.RowsAffected,
	})
}
