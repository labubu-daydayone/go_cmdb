package dns

import (
	"net/http"
	"strconv"

	"go_cmdb/internal/dns"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles DNS record API requests
type Handler struct {
	db      *gorm.DB
	service *dns.Service
}

// NewHandler creates a new DNS handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:      db,
		service: dns.NewService(db),
	}
}

// CreateRecordRequest represents the request body for creating a DNS record
type CreateRecordRequest struct {
	DomainID  int                        `json:"domainId" binding:"required"`
	Type      model.DNSRecordType        `json:"type" binding:"required,oneof=A AAAA CNAME TXT"`
	Name      string                     `json:"name" binding:"required"`
	Value     string                     `json:"value" binding:"required"`
	TTL       int                        `json:"ttl"`
	OwnerType model.DNSRecordOwnerType   `json:"ownerType" binding:"required,oneof=node_group line_group website_domain acme_challenge"`
	OwnerID   int                        `json:"ownerId" binding:"required"`
}

// CreateRecord creates a new DNS record
// POST /api/v1/dns/records/create
func (h *Handler) CreateRecord(c *gin.Context) {
	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	// Set default TTL if not provided
	if req.TTL == 0 {
		req.TTL = 120
	}

	// Create DNS record (status=pending, desired_state=present)
	record := model.DomainDNSRecord{
		DomainID:     req.DomainID,
		Type:         req.Type,
		Name:         req.Name,
		Value:        req.Value,
		TTL:          req.TTL,
		Proxied:      false, // Default to DNS only (not proxied)
		Status:       model.DNSRecordStatusPending,
		DesiredState: model.DNSRecordDesiredStatePresent,
		OwnerType:    req.OwnerType,
		OwnerID:      req.OwnerID,
	}

	if err := h.db.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to create DNS record: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    record,
	})
}

// DeleteRecordRequest represents the request body for deleting DNS records
type DeleteRecordRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// DeleteRecord marks DNS records for deletion (desired_state=absent)
// POST /api/v1/dns/records/delete
func (h *Handler) DeleteRecord(c *gin.Context) {
	var req DeleteRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	// Mark records as absent (worker will delete them)
	result := h.db.Model(&model.DomainDNSRecord{}).
		Where("id IN ?", req.IDs).
		Update("desired_state", model.DNSRecordDesiredStateAbsent)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to mark records for deletion: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"affected": result.RowsAffected,
		},
	})
}

// RetryRecordRequest represents the request body for retrying a DNS record
type RetryRecordRequest struct {
	ID int `json:"id" binding:"required"`
}

// RetryRecord resets retry state for a DNS record (manual retry)
// POST /api/v1/dns/records/retry
func (h *Handler) RetryRecord(c *gin.Context) {
	var req RetryRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	// Reset retry state (only for error status)
	if err := h.service.ResetRetry(req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to reset retry: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// ListRecordsQuery represents the query parameters for listing DNS records
type ListRecordsQuery struct {
	DomainID  int                        `form:"domainId"`
	Status    model.DNSRecordStatus      `form:"status"`
	OwnerType model.DNSRecordOwnerType   `form:"ownerType"`
	OwnerID   int                        `form:"ownerId"`
	Page      int                        `form:"page"`
	PageSize  int                        `form:"pageSize"`
}

// ListRecords lists DNS records with filters and pagination
// GET /api/v1/dns/records
func (h *Handler) ListRecords(c *gin.Context) {
	var query ListRecordsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid query: " + err.Error(),
		})
		return
	}

	// Set default pagination
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}

	// Build query
	db := h.db.Model(&model.DomainDNSRecord{})

	if query.DomainID > 0 {
		db = db.Where("domain_id = ?", query.DomainID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OwnerType != "" {
		db = db.Where("owner_type = ?", query.OwnerType)
	}
	if query.OwnerID > 0 {
		db = db.Where("owner_id = ?", query.OwnerID)
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to count records: " + err.Error(),
		})
		return
	}

	// Query records
	var records []model.DomainDNSRecord
	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Order("id DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to query records: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     records,
			"total":    total,
			"page":     query.Page,
			"pageSize": query.PageSize,
		},
	})
}

// GetRecord retrieves a single DNS record by ID
// GET /api/v1/dns/records/:id
func (h *Handler) GetRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid record ID",
		})
		return
	}

	var record model.DomainDNSRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "record not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to query record: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    record,
	})
}

// SyncRecordsRequest represents the request body for syncing DNS records
type SyncRecordsRequest struct {
	DomainID int `json:"domainId" binding:"required"`
}

// SyncRecords pulls DNS records from Cloudflare and syncs to local database
// POST /api/v1/dns/records/sync
func (h *Handler) SyncRecords(c *gin.Context) {
	var req SyncRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    2001,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	result, err := dns.PullSyncRecords(c.Request.Context(), req.DomainID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    3001,
			"message": "sync failed: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
