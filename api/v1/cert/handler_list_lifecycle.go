package cert

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

// CertificateLifecycleItem represents a unified certificate lifecycle item
type CertificateLifecycleItem struct {
	ID            string     `json:"id"` // Unique string identifier: "cert:<id>" or "req:<id>" (display-only, not a business key)
	ItemType      string     `json:"itemType"` // "certificate" | "request"
	CertificateID *int       `json:"certificateId"`
	RequestID     *int       `json:"requestId"`
	Status        string     `json:"status"` // issuing|failed|valid|expiring|expired|revoked (NO "issued")
	Domains       []string   `json:"domains"`
	Fingerprint   string     `json:"fingerprint"`
	IssueAt       *time.Time `json:"issueAt"`
	ExpireAt      *time.Time `json:"expireAt"`
	LastError     *string    `json:"lastError"`
	CreatedAt     *time.Time `json:"createdAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
	Deletable     bool       `json:"deletable"` // Whether this item can be deleted
}

// normalizeDomainSetKey creates a normalized key from domains for deduplication
// Converts to lowercase, trims spaces, removes duplicates, sorts, and joins with comma
func normalizeDomainSetKey(domains []string) string {
	normalized := make([]string, 0, len(domains))
	seen := make(map[string]bool)
	
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" && !seen[d] {
			normalized = append(normalized, d)
			seen[d] = true
		}
	}
	
	sort.Strings(normalized)
	return strings.Join(normalized, ",")
}

// ListCertificatesLifecycle lists all certificates and certificate requests (unified lifecycle view with deduplication)
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

	// 1. Query all certificates with their domains
	var certificates []model.Certificate
	certQuery := h.db.Model(&model.Certificate{})
	
	if source != "" {
		certQuery = certQuery.Where("source = ?", source)
	}
	if provider != "" {
		certQuery = certQuery.Where("provider = ?", provider)
	}
	if status != "" && status != "issuing" && status != "failed" {
		// Only apply certificate status filters (valid/expiring/expired/revoked)
		certQuery = certQuery.Where("status = ?", status)
	}
	
	certQuery.Order("created_at DESC").Find(&certificates)

	// Build certificate map by domain_set_key
	certMap := make(map[string]*model.Certificate)
	certDomainsMap := make(map[int][]string) // certificateId -> domains
	
	for i := range certificates {
		cert := &certificates[i]
		
		// Query domains for this certificate
		var certDomains []model.CertificateDomain
		h.db.Where("certificate_id = ?", cert.ID).Find(&certDomains)
		
		domains := make([]string, len(certDomains))
		for j, cd := range certDomains {
			domains[j] = cd.Domain
		}
		
		certDomainsMap[cert.ID] = domains
		key := normalizeDomainSetKey(domains)
		
		// Priority: valid/expiring/expired/revoked certificates take precedence
		if existing, ok := certMap[key]; !ok || existing.CreatedAt.Before(*cert.CreatedAt) {
			certMap[key] = cert
		}
	}

	// 2. Query all certificate requests
	var requests []model.CertificateRequest
	requestQuery := h.db.Model(&model.CertificateRequest{})
	
	if status != "" {
		switch status {
		case "issuing":
			requestQuery = requestQuery.Where("status IN ?", []string{"pending", "running", "success"})
		case "failed":
			requestQuery = requestQuery.Where("status = ?", "failed")
		}
	}
	
	requestQuery.Order("created_at DESC").Find(&requests)

	// Build request map by domain_set_key
	requestMap := make(map[string]*model.CertificateRequest)
	requestDomainsMap := make(map[int][]string) // requestId -> domains
	
	for i := range requests {
		req := &requests[i]
		
		var domains []string
		if err := json.Unmarshal([]byte(req.Domains), &domains); err != nil {
			domains = []string{}
		}
		
		requestDomainsMap[req.ID] = domains
		key := normalizeDomainSetKey(domains)
		
		// Only keep latest request for each domain set
		if existing, ok := requestMap[key]; !ok || existing.CreatedAt.Before(*req.CreatedAt) {
			requestMap[key] = req
		}
	}

	// 3. Merge with priority: certificate > request
	mergedMap := make(map[string]CertificateLifecycleItem)
	
	// First, add all certificates (highest priority)
	for key, cert := range certMap {
		domains := certDomainsMap[cert.ID]
		
		// Check if certificate has active bindings
		var activeBindingCount int64
		h.db.Model(&model.CertificateBinding{}).
			Where("certificate_id = ? AND is_active = ?", cert.ID, true).
			Count(&activeBindingCount)
		
		item := CertificateLifecycleItem{
			ID:            "cert:" + strconv.Itoa(cert.ID),
			ItemType:      "certificate",
			CertificateID: &cert.ID,
			RequestID:     nil,
			Status:        cert.Status, // valid/expiring/expired/revoked
			Domains:       domains,
			Fingerprint:   cert.Fingerprint,
			IssueAt:       cert.IssueAt,
			ExpireAt:      cert.ExpireAt,
			LastError:     cert.LastError,
			CreatedAt:     cert.CreatedAt,
			UpdatedAt:     cert.UpdatedAt,
			Deletable:     activeBindingCount == 0,
		}
		
		mergedMap[key] = item
	}
	
	// Then, add requests only if no certificate exists for that domain set
	for key, req := range requestMap {
		if _, exists := mergedMap[key]; exists {
			// Certificate already exists, skip request
			continue
		}
		
		domains := requestDomainsMap[req.ID]
		
		// Map status: pending/running/success -> issuing, failed -> failed
		var mappedStatus string
		switch req.Status {
		case "pending", "running", "success":
			mappedStatus = "issuing" // NO "issued" status in output
		case "failed":
			mappedStatus = "failed"
		default:
			mappedStatus = "issuing"
		}
		
		item := CertificateLifecycleItem{
			ID:            "req:" + strconv.Itoa(req.ID),
			ItemType:      "request",
			CertificateID: nil,
			RequestID:     &req.ID,
			Status:        mappedStatus,
			Domains:       domains,
			Fingerprint:   "",
			IssueAt:       nil,
			ExpireAt:      nil,
			LastError:     req.LastError,
			CreatedAt:     req.CreatedAt,
			UpdatedAt:     req.UpdatedAt,
			Deletable:     mappedStatus == "failed",
		}
		
		mergedMap[key] = item
	}

	// 4. Convert map to slice and sort by createdAt DESC
	items := make([]CertificateLifecycleItem, 0, len(mergedMap))
	for _, item := range mergedMap {
		items = append(items, item)
	}
	
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == nil {
			return false
		}
		if items[j].CreatedAt == nil {
			return true
		}
		return items[i].CreatedAt.After(*items[j].CreatedAt)
	})

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

	httpx.OKItems(c, items, int64(total), page, pageSize)
}
