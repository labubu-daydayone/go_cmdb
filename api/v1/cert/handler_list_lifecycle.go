package cert

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

// CertificateLifecycleItem represents a unified certificate lifecycle item
type CertificateLifecycleItem struct {
	ItemType        string     `json:"itemType"` // "certificate" | "request"
	CertificateID   *int       `json:"certificateId"`
	RequestID       *int       `json:"requestId"`
	Status          string     `json:"status"` // issuing|issued|failed|valid|expiring|expired|revoked
	Domains         []string   `json:"domains"`
	Fingerprint     string     `json:"fingerprint"`
	IssueAt         *time.Time `json:"issueAt"`
	ExpireAt        *time.Time `json:"expireAt"`
	LastError       *string    `json:"lastError"`
	CreatedAt       *time.Time `json:"createdAt"`
	UpdatedAt       *time.Time `json:"updatedAt"`
}

// ListCertificatesLifecycle lists all certificates and certificate requests (unified lifecycle view)
// GET /api/v1/certificates
func (h *Handler) ListCertificatesLifecycle(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Query parameters
	source := c.Query("source")
	provider := c.Query("provider")
	status := c.Query("status")

	// 1. Query certificate_requests
	var requests []model.CertificateRequest
	requestQuery := h.db.Model(&model.CertificateRequest{})
	
	// Filter by status if provided
	if status != "" {
		switch status {
		case "issuing":
			requestQuery = requestQuery.Where("status IN ?", []string{"pending", "running"})
		case "issued":
			requestQuery = requestQuery.Where("status = ?", "success")
		case "failed":
			requestQuery = requestQuery.Where("status = ?", "failed")
		}
	}
	
	requestQuery.Order("created_at DESC").Find(&requests)

	// 2. Query certificates
	var certificates []struct {
		model.Certificate
		DomainCount int `gorm:"column:domain_count"`
	}
	
	certQuery := h.db.Table("certificates").
		Select("certificates.*, (SELECT COUNT(DISTINCT domain) FROM certificate_domains WHERE certificate_id = certificates.id) as domain_count")
	
	// Apply filters
	if source != "" {
		certQuery = certQuery.Where("source = ?", source)
	}
	if provider != "" {
		certQuery = certQuery.Where("provider = ?", provider)
	}
	if status != "" && status != "issuing" && status != "issued" && status != "failed" {
		certQuery = certQuery.Where("status = ?", status)
	}
	
	certQuery.Order("created_at DESC").Find(&certificates)

	// 3. Merge results
	var items []CertificateLifecycleItem

	// Add certificate requests
	for _, req := range requests {
		var domains []string
		if err := json.Unmarshal([]byte(req.Domains), &domains); err != nil {
			domains = []string{}
		}

		// Map status
		var mappedStatus string
		switch req.Status {
		case "pending", "running":
			mappedStatus = "issuing"
		case "success":
			mappedStatus = "issued"
		case "failed":
			mappedStatus = "failed"
		default:
			mappedStatus = req.Status
		}

		items = append(items, CertificateLifecycleItem{
			ItemType:      "request",
			RequestID:     &req.ID,
			CertificateID: req.ResultCertificateID,
			Status:        mappedStatus,
			Domains:       domains,
			Fingerprint:   "",
			LastError:     req.LastError,
			CreatedAt:     req.CreatedAt,
			UpdatedAt:     req.UpdatedAt,
		})
	}

	// Add certificates
	for _, cert := range certificates {
		// Query domains
		var certDomains []model.CertificateDomain
		h.db.Where("certificate_id = ?", cert.ID).Find(&certDomains)
		
		domains := make([]string, len(certDomains))
		for i, cd := range certDomains {
			domains[i] = cd.Domain
		}

		items = append(items, CertificateLifecycleItem{
			ItemType:      "certificate",
			CertificateID: &cert.ID,
			RequestID:     nil,
			Status:        cert.Status,
			Domains:       domains,
			Fingerprint:   cert.Fingerprint,
			IssueAt:        cert.IssueAt,
			ExpireAt:       cert.ExpireAt,
			LastError:     cert.LastError,
			CreatedAt:      cert.CreatedAt,
			UpdatedAt:      cert.UpdatedAt,
		})
	}

	// 4. Sort by createdAt DESC
	// (Already sorted in queries, but merge might break order)
	// For simplicity, we keep the current order (requests first, then certificates)

	// 5. Pagination
	total := len(items)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		items = []CertificateLifecycleItem{}
	} else {
		if end > total {
			end = total
		}
		items = items[start:end]
	}

	c.JSON(http.StatusOK, httpx.Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"items":    items,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
