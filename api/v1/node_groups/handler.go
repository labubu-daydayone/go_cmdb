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
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CNAMEPrefix string `json:"cnamePrefix"`

	Status      string `json:"status"`
	IPCount     int    `json:"ipCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreateRequest represents create node group request
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IpIds       []int  `json:"ipIds" binding:"required,min=1"`
}

// UpdateRequest represents update node group request
type UpdateRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	IpIds       []int   `json:"ipIds"`
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
				DomainID:     domain.ID,
				Type:         model.DNSRecordTypeA,
				Name:         nodeGroup.CNAMEPrefix,
				Value:        ip.IP,
				TTL:          120,
				OwnerType:    "node_group",
				OwnerID:      nodeGroup.ID,
				Status:       model.DNSRecordStatusPending,
				DesiredState: model.DNSRecordDesiredStatePresent,
			}

			// Use raw SQL for upsert to handle unique constraint properly
			sql := `INSERT INTO domain_dns_records (domain_id, type, name, value, ttl, owner_type, owner_id, status, desired_state, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
				ON DUPLICATE KEY UPDATE desired_state = VALUES(desired_state), status = VALUES(status), updated_at = NOW()`
			if err := tx.Exec(sql, dnsRecord.DomainID, dnsRecord.Type, dnsRecord.Name, dnsRecord.Value, dnsRecord.TTL,
				dnsRecord.OwnerType, dnsRecord.OwnerID, dnsRecord.Status, dnsRecord.DesiredState).Error; err != nil {
				return fmt.Errorf("failed to create DNS record for domain %s, IP %s: %w", domain.Domain, ip.IP, err)
			}
		}
	}

	return nil
}

// markDNSRecordsAsAbsent marks DNS records as absent for a node group
func (h *Handler) markDNSRecordsAsAbsent(tx *gorm.DB, nodeGroupID int) error {
	updates := map[string]interface{}{
		"desired_state": model.DNSRecordDesiredStateAbsent,
	}

	if err := tx.Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id = ?", "node_group", nodeGroupID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark DNS records as absent: %w", err)
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
			ID:          ng.ID,
			Name:        ng.Name,
			Description: ng.Description,
			CNAMEPrefix: ng.CNAMEPrefix,
			Status:      string(ng.Status),
			IPCount:     int(count),
			CreatedAt:   ng.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   ng.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

	// Use first CDN domain for CNAME display
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

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	nodeGroup := model.NodeGroup{
		Name:        req.Name,
		Description: req.Description,
		CNAMEPrefix: cnamePrefix,
		Status:      model.NodeGroupStatusActive,
	}

	if err := tx.Create(&nodeGroup).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create node group", err))
		return
	}

	if len(req.IpIds) > 0 {
		for _, ipID := range req.IpIds {
			mapping := model.NodeGroupIP{
				NodeGroupID: nodeGroup.ID,
				IPID:        ipID,
			}
			if err := tx.Create(&mapping).Error; err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP mapping", err))
				return
			}
		}

		// Create DNS records for all CDN domains × all available IPs
		if err := h.createDNSRecordsForAllCDNDomains(tx, &nodeGroup, cdnDomains, req.IpIds); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS records", err))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	// Reload node group
	if err := h.db.First(&nodeGroup, nodeGroup.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload node group", err))
		return
	}

	// Return response with data.item structure
	httpx.OK(c, map[string]interface{}{
		"item": map[string]interface{}{
			"id":          nodeGroup.ID,
			"name":        nodeGroup.Name,
			"description": nodeGroup.Description,
			"cnamePrefix": nodeGroup.CNAMEPrefix,
			"status":      nodeGroup.Status,
			"createdAt":   nodeGroup.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updatedAt":   nodeGroup.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
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

	// Handle IP updates
	if req.IpIds != nil {
		// Mark old DNS records as absent
		if err := h.markDNSRecordsAsAbsent(tx, req.ID); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark old DNS records as absent", err))
			return
		}

		// Delete old IP mappings
		if err := tx.Where("node_group_id = ?", req.ID).Delete(&model.NodeGroupIP{}).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete old sub IP mappings", err))
			return
		}

		// Create new IP mappings and DNS records
		if len(req.IpIds) > 0 {
			for _, ipID := range req.IpIds {
				mapping := model.NodeGroupIP{
					NodeGroupID: req.ID,
					IPID:        ipID,
				}
				if err := tx.Create(&mapping).Error; err != nil {
					tx.Rollback()
					httpx.FailErr(c, httpx.ErrDatabaseError("failed to create sub IP mapping", err))
					return
				}
			}

			// Fetch all CDN domains
			var cdnDomains []model.Domain
			if err := tx.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch CDN domains", err))
				return
			}

			// Create new DNS records for all CDN domains
			if err := h.createDNSRecordsForAllCDNDomains(tx, &nodeGroup, cdnDomains, req.IpIds); err != nil {
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

	// Reload node group
	if err := h.db.First(&nodeGroup, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload node group", err))
		return
	}

	// Return response with data.item structure
	httpx.OK(c, map[string]interface{}{
		"item": map[string]interface{}{
			"id":          nodeGroup.ID,
			"name":        nodeGroup.Name,
			"description": nodeGroup.Description,
			"cnamePrefix": nodeGroup.CNAMEPrefix,
			"status":      nodeGroup.Status,
			"createdAt":   nodeGroup.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updatedAt":   nodeGroup.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
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

	// Mark all DNS records as absent for each node group
	for _, id := range req.IDs {
		if err := h.markDNSRecordsAsAbsent(tx, id); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark DNS records as absent", err))
			return
		}
	}

	// Delete node groups (cascade will delete node_group_ips)
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

	// Return null data for delete operation
	httpx.OK(c, nil)
}
