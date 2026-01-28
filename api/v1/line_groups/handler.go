package line_groups

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRequest represents list line groups request
type ListRequest struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	Name        string `form:"name"`
	DomainID    *int   `form:"domainId"`
	NodeGroupID *int   `form:"nodeGroupId"`
	Status      string `form:"status"`
}

// ListResponse represents list line groups response
type ListResponse struct {
	Items    []LineGroupItemDTO `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// CreateRequest represents create line group request
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	DomainID    int    `json:"domainId" binding:"required"`
	NodeGroupID int    `json:"nodeGroupId" binding:"required"`
}

// UpdateRequest represents update line group request
type UpdateRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        *string `json:"name"`
	Status      *string `json:"status"`
	NodeGroupID *int    `json:"nodeGroupId"`
}

// DeleteRequest represents delete line groups request
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Handler handles line groups API
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new line groups handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// generateCNAMEPrefix generates a random CNAME prefix
func generateCNAMEPrefix() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "lg-" + hex.EncodeToString(bytes)
}

// createDNSRecordForLineGroup creates CNAME record for line group
func (h *Handler) createDNSRecordForLineGroup(tx *gorm.DB, lineGroup *model.LineGroup, nodeGroupCNAME string) error {
	dnsRecord := model.DomainDNSRecord{
		DomainID:  int(lineGroup.DomainID),
		Type:      model.DNSRecordTypeCNAME,
		Name:      lineGroup.CNAMEPrefix,
		Value:     nodeGroupCNAME,
		OwnerType: "line_group",
		OwnerID:   int(lineGroup.ID),
		Status:    model.DNSRecordStatusPending,
	}

	if err := tx.Create(&dnsRecord).Error; err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	return nil
}

// markDNSRecordsForDeletion marks DNS records for deletion when node group changes
func (h *Handler) markDNSRecordsForDeletion(tx *gorm.DB, lineGroupID int, reason string) error {
	updates := map[string]interface{}{
		"desired_state": model.DNSRecordStateAbsent,
		"status":        model.DNSRecordStatusPending,
		"last_error":    reason,
	}

	if err := tx.Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id = ? AND desired_state = ?", "line_group", lineGroupID, model.DNSRecordStatePresent).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark DNS records for deletion: %w", err)
	}

	return nil
}

// List handles GET /api/v1/line-groups
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
	query := h.db.Model(&model.LineGroup{})

	// Name filter (fuzzy)
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// DomainID filter
	if req.DomainID != nil {
		query = query.Where("domain_id = ?", *req.DomainID)
	}

	// NodeGroupID filter
	if req.NodeGroupID != nil {
		query = query.Where("node_group_id = ?", *req.NodeGroupID)
	}

	// Status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count line groups", err))
		return
	}

	// Fetch line groups with pagination
	var lineGroups []model.LineGroup
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Preload("Domain").
		Preload("NodeGroup").
		Offset(offset).
		Limit(req.PageSize).
		Order("id DESC").
		Find(&lineGroups).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch line groups", err))
		return
	}

	// Convert to DTOs
	items := make([]LineGroupItemDTO, len(lineGroups))
	for i, lg := range lineGroups {
		domainName := ""
		if lg.Domain != nil {
			domainName = lg.Domain.Domain
		}
		
		nodeGroupName := ""
		if lg.NodeGroup != nil {
			nodeGroupName = lg.NodeGroup.Name
		}
		
		items[i] = LineGroupItemDTO{
			ID:            int(lg.ID),
			Name:          lg.Name,
			DomainID:      int(lg.DomainID),
			DomainName:    domainName,
			NodeGroupID:   int(lg.NodeGroupID),
			NodeGroupName: nodeGroupName,
			CNAMEPrefix:   lg.CNAMEPrefix,
			CNAME:         lg.CNAMEPrefix + "." + domainName,
			Status:        lg.Status,
			CreatedAt:     lg.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
			UpdatedAt:     lg.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
		}
	}

	httpx.OK(c, ListResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create handles POST /api/v1/line-groups/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Check domain exists
	var domain model.Domain
	if err := h.db.First(&domain, req.DomainID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("domain not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find domain", err))
		return
	}

	// Check node group exists
	var nodeGroup model.NodeGroup
	if err := h.db.First(&nodeGroup, req.NodeGroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("node group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node group", err))
		return
	}

	// Check name uniqueness
	var count int64
	if err := h.db.Model(&model.LineGroup{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if count > 0 {
		httpx.FailErr(c, httpx.ErrAlreadyExists("line group name already exists"))
		return
	}

	// Generate CNAME prefix
	var cnamePrefix string
	for {
		cnamePrefix = generateCNAMEPrefix()
		var existCount int64
		if err := h.db.Model(&model.LineGroup{}).Where("cname_prefix = ?", cnamePrefix).Count(&existCount).Error; err != nil {
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

	lineGroup := model.LineGroup{
		Name:        req.Name,
		DomainID:    int64(req.DomainID),
		NodeGroupID: int64(req.NodeGroupID),
		CNAMEPrefix: cnamePrefix,
		Status:      model.LineGroupStatusActive,
	}

	if err := tx.Create(&lineGroup).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create line group", err))
		return
	}

	// Create DNS CNAME record with full FQDN
	nodeGroupCNAME := nodeGroup.CNAMEPrefix + "." + domain.Domain
	if err := h.createDNSRecordForLineGroup(tx, &lineGroup, nodeGroupCNAME); err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS record", err))
		return
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	if err := h.db.Preload("Domain").Preload("NodeGroup").First(&lineGroup, lineGroup.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload line group", err))
		return
	}

	// Convert to DTO
	domainName := ""
	if lineGroup.Domain != nil {
		domainName = lineGroup.Domain.Domain
	}
	
	nodeGroupName := ""
	if lineGroup.NodeGroup != nil {
		nodeGroupName = lineGroup.NodeGroup.Name
	}
	
	item := LineGroupItemDTO{
		ID:            int(lineGroup.ID),
		Name:          lineGroup.Name,
		DomainID:      int(lineGroup.DomainID),
		DomainName:    domainName,
		NodeGroupID:   int(lineGroup.NodeGroupID),
		NodeGroupName: nodeGroupName,
		CNAMEPrefix:   lineGroup.CNAMEPrefix,
		CNAME:         lineGroup.CNAMEPrefix + "." + domainName,
		Status:        lineGroup.Status,
		CreatedAt:     lineGroup.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		UpdatedAt:     lineGroup.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
	}

	httpx.OK(c, gin.H{"item": item})
}

// Update handles POST /api/v1/line-groups/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	var lineGroup model.LineGroup
	if err := h.db.First(&lineGroup, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("line group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find line group", err))
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
		if err := tx.Model(&model.LineGroup{}).Where("name = ? AND id != ?", *req.Name, req.ID).Count(&count).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
			return
		}
		if count > 0 {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrAlreadyExists("line group name already exists"))
			return
		}
		updates["name"] = *req.Name
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if req.NodeGroupID != nil {
		// Check node group exists
		var nodeGroup model.NodeGroup
		if err := tx.First(&nodeGroup, *req.NodeGroupID).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				httpx.FailErr(c, httpx.ErrNotFound("node group not found"))
				return
			}
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node group", err))
			return
		}

		// Mark old DNS records for deletion
		if err := h.markDNSRecordsForDeletion(tx, req.ID, "node group changed"); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark old DNS records for deletion", err))
			return
		}

		updates["node_group_id"] = *req.NodeGroupID

		// Load domain to construct full FQDN
		var domain model.Domain
		if err := tx.First(&domain, lineGroup.DomainID).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find domain", err))
			return
		}

		// Create new DNS CNAME record with full FQDN
		lineGroup.NodeGroupID = int64(*req.NodeGroupID)
		nodeGroupCNAME := nodeGroup.CNAMEPrefix + "." + domain.Domain
		if err := h.createDNSRecordForLineGroup(tx, &lineGroup, nodeGroupCNAME); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS record", err))
			return
		}
	}

	if len(updates) > 0 {
		if err := tx.Model(&lineGroup).Updates(updates).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to update line group", err))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	if err := h.db.Preload("Domain").Preload("NodeGroup").First(&lineGroup, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload line group", err))
		return
	}

	// Convert to DTO
	domainName := ""
	if lineGroup.Domain != nil {
		domainName = lineGroup.Domain.Domain
	}
	
	nodeGroupName := ""
	if lineGroup.NodeGroup != nil {
		nodeGroupName = lineGroup.NodeGroup.Name
	}
	
	item := LineGroupItemDTO{
		ID:            int(lineGroup.ID),
		Name:          lineGroup.Name,
		DomainID:      int(lineGroup.DomainID),
		DomainName:    domainName,
		NodeGroupID:   int(lineGroup.NodeGroupID),
		NodeGroupName: nodeGroupName,
		CNAMEPrefix:   lineGroup.CNAMEPrefix,
		CNAME:         lineGroup.CNAMEPrefix + "." + domainName,
		Status:        lineGroup.Status,
		CreatedAt:     lineGroup.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		UpdatedAt:     lineGroup.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
	}

	httpx.OK(c, gin.H{"item": item})
}

// Delete handles POST /api/v1/line-groups/delete
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
		if err := h.markDNSRecordsForDeletion(tx, id, "line group deleted"); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark DNS records for deletion", err))
			return
		}
	}

	result := tx.Delete(&model.LineGroup{}, req.IDs)
	if result.Error != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete line groups", result.Error))
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

// RepairCNAMERequest represents repair CNAME request
type RepairCNAMERequest struct {
	LineGroupID int `json:"lineGroupId" binding:"required"`
}

// RepairCNAMEResponse represents repair CNAME response
type RepairCNAMEResponse struct {
	LineGroupID   int    `json:"lineGroupId"`
	DomainID      int    `json:"domainId"`
	Domain        string `json:"domain"`
	ExpectedValue string `json:"expectedValue"`
	Affected      int    `json:"affected"`
}

// RepairCNAME handles POST /api/v1/line-groups/dns/repair-cname
func (h *Handler) RepairCNAME(c *gin.Context) {
	var req RepairCNAMERequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
		return
	}

	// Load line group
	var lineGroup model.LineGroup
	if err := h.db.First(&lineGroup, req.LineGroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("line group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find line group", err))
		return
	}

	// Load node group
	var nodeGroup model.NodeGroup
	if err := h.db.First(&nodeGroup, lineGroup.NodeGroupID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find node group", err))
		return
	}

	// Load domain
	var domain model.Domain
	if err := h.db.First(&domain, lineGroup.DomainID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find domain", err))
		return
	}

	// Calculate expected CNAME value
	expectedValue := nodeGroup.CNAMEPrefix + "." + domain.Domain

	// Find and update incorrect DNS records
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var dnsRecords []model.DomainDNSRecord
	if err := tx.Where("owner_type = ? AND owner_id = ? AND type = ?", 
		"line_group", lineGroup.ID, model.DNSRecordTypeCNAME).
		Find(&dnsRecords).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query DNS records", err))
		return
	}

	affected := 0
	for _, record := range dnsRecords {
		// Check if value is incorrect (missing domain suffix or ends with just ".")
		if record.Value != expectedValue {
			// Mark old record for deletion
			updates := map[string]interface{}{
				"status":        model.DNSRecordStatusPending,
				"desired_state": model.DNSRecordDesiredStateAbsent,
			}
			if err := tx.Model(&model.DomainDNSRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark old record for deletion", err))
				return
			}
			affected++
		}
	}

	// Create new DNS record with correct value
	if affected > 0 {
		if err := h.createDNSRecordForLineGroup(tx, &lineGroup, expectedValue); err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create new DNS record", err))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	response := RepairCNAMEResponse{
		LineGroupID:   int(lineGroup.ID),
		DomainID:      int(lineGroup.DomainID),
		Domain:        domain.Domain,
		ExpectedValue: expectedValue,
		Affected:      affected,
	}

	httpx.OK(c, gin.H{
		"item": response,
	})
}
