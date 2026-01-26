package cert

import (
	"go_cmdb/internal/cert"
	"go_cmdb/internal/httpx"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles certificate-related API requests
type Handler struct {
	db          *gorm.DB
	certService *cert.Service
}

// NewHandler creates a new certificate handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:          db,
		certService: cert.NewService(db),
	}
}

// GetCertificateWebsites handles GET /api/v1/certificates/{id}/websites
func (h *Handler) GetCertificateWebsites(c *gin.Context) {
	// Parse certificate ID
	certIDStr := c.Param("id")
	certID, err := strconv.Atoi(certIDStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid certificate ID"))
		return
	}

	// Check if certificate exists
	var certExists bool
	if err := h.db.Raw("SELECT EXISTS(SELECT 1 FROM certificates WHERE id = ?)", certID).Scan(&certExists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to check certificate", err))
		return
	}

	if !certExists {
		httpx.FailErr(c, httpx.ErrNotFound("Certificate not found"))
		return
	}

	// Get certificate domains
	certDomains, err := h.certService.GetCertificateDomains(certID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get certificate domains", err))
		return
	}

	// Get websites using this certificate
	websites, err := h.certService.GetCertificateWebsites(certID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get certificate websites", err))
		return
	}

	httpx.OK(c, gin.H{
		"certificateId": certID,
		"domains":       certDomains,
		"websites":      websites,
	})
}

// GetWebsiteCertificateCandidates handles GET /api/v1/websites/{id}/certificates/candidates
func (h *Handler) GetWebsiteCertificateCandidates(c *gin.Context) {
	// Parse website ID
	websiteIDStr := c.Param("id")
	websiteID, err := strconv.Atoi(websiteIDStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid website ID"))
		return
	}

	// Check if website exists
	var websiteExists bool
	if err := h.db.Raw("SELECT EXISTS(SELECT 1 FROM websites WHERE id = ?)", websiteID).Scan(&websiteExists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to check website", err))
		return
	}

	if !websiteExists {
		httpx.FailErr(c, httpx.ErrNotFound("Website not found"))
		return
	}

	// Get website domains
	websiteDomains, err := h.certService.GetWebsiteDomains(websiteID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get website domains", err))
		return
	}

	// Get certificate candidates
	candidates, err := h.certService.GetWebsiteCertificateCandidates(websiteID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get certificate candidates", err))
		return
	}

	httpx.OK(c, gin.H{
		"websiteId":      websiteID,
		"websiteDomains": websiteDomains,
		"candidates":     candidates,
	})
}

// ListCertificates handles GET /api/v1/certificates?page=1&pageSize=20&source=&provider=&status=
func (h *Handler) ListCertificates(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	source := c.Query("source")       // manual|acme
	provider := c.Query("provider")   // acme provider name (for filtering acme certificates)
	status := c.Query("status")       // pending|issued|expired|revoked|valid|expiring

	// Build query with subquery to get domain count
	query := h.db.Table("certificates c").
		Select(`
			c.id,
			c.name,
			c.fingerprint,
			c.status,
			c.source,
			c.renew_mode,
			c.issuer,
			c.issue_at,
			c.expire_at,
			c.acme_account_id,
			c.renewing,
			c.last_error,
			c.created_at,
			c.updated_at,
			(SELECT COUNT(*) FROM certificate_domains WHERE certificate_id = c.id) as domain_count
		`)

	// Apply filters
	if source != "" {
		query = query.Where("c.source = ?", source)
	}
	if status != "" {
		query = query.Where("c.status = ?", status)
	}
	// Provider filter: match acme_account_id with acme_accounts.provider_id
	if provider != "" {
		query = query.Where(`c.acme_account_id IN (
			SELECT id FROM acme_accounts WHERE provider_id = (
				SELECT id FROM acme_providers WHERE name = ?
			)
		)`, provider)
	}

	// Count total (before pagination)
	var total int64
	countQuery := h.db.Table("certificates c")
	if source != "" {
		countQuery = countQuery.Where("c.source = ?", source)
	}
	if status != "" {
		countQuery = countQuery.Where("c.status = ?", status)
	}
	if provider != "" {
		countQuery = countQuery.Where(`c.acme_account_id IN (
			SELECT id FROM acme_accounts WHERE provider_id = (
				SELECT id FROM acme_providers WHERE name = ?
			)
		)`, provider)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to count certificates", err))
		return
	}

	// Query with pagination
	offset := (page - 1) * pageSize
	query = query.Order("c.id DESC").Limit(pageSize).Offset(offset)

	type QueryRow struct {
		ID            int    `gorm:"column:id"`
		Name          string `gorm:"column:name"`
		Fingerprint   string `gorm:"column:fingerprint"`
		Status        string `gorm:"column:status"`
		Source        string `gorm:"column:source"`
		RenewMode     string `gorm:"column:renew_mode"`
		Issuer        string `gorm:"column:issuer"`
		IssueAt       string `gorm:"column:issue_at"`
		ExpireAt      string `gorm:"column:expire_at"`
		AcmeAccountID int    `gorm:"column:acme_account_id"`
		Renewing      bool   `gorm:"column:renewing"`
		LastError     *string `gorm:"column:last_error"`
		CreatedAt     string `gorm:"column:created_at"`
		UpdatedAt     string `gorm:"column:updated_at"`
		DomainCount   int    `gorm:"column:domain_count"`
	}

	var rows []QueryRow
	if err := query.Scan(&rows).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to query certificates", err))
		return
	}

	// Build response items
	type CertificateItem struct {
		ID            int     `json:"id"`
		Name          string  `json:"name"`
		Fingerprint   string  `json:"fingerprint"`
		Status        string  `json:"status"`
		Source        string  `json:"source"`
		RenewMode     string  `json:"renewMode"`
		Issuer        string  `json:"issuer"`
		IssueAt       string  `json:"issueAt"`
		ExpireAt      string  `json:"expireAt"`
		AcmeAccountID int     `json:"acmeAccountId"`
		Renewing      bool    `json:"renewing"`
		LastError     *string `json:"lastError"`
		DomainCount   int     `json:"domainCount"`
		CreatedAt     string  `json:"createdAt"`
		UpdatedAt     string  `json:"updatedAt"`
	}

	items := make([]CertificateItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, CertificateItem{
			ID:            row.ID,
			Name:          row.Name,
			Fingerprint:   row.Fingerprint,
			Status:        row.Status,
			Source:        row.Source,
			RenewMode:     row.RenewMode,
			Issuer:        row.Issuer,
			IssueAt:       row.IssueAt,
			ExpireAt:      row.ExpireAt,
			AcmeAccountID: row.AcmeAccountID,
			Renewing:      row.Renewing,
			LastError:     row.LastError,
			DomainCount:   row.DomainCount,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}

	httpx.OKItems(c, items, total, page, pageSize)
}

// GetCertificate handles GET /api/v1/certificates/:id
func (h *Handler) GetCertificate(c *gin.Context) {
	// Parse certificate ID
	certIDStr := c.Param("id")
	certID, err := strconv.Atoi(certIDStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid certificate ID"))
		return
	}

	// Query certificate with full details
	type CertificateDetail struct {
		ID            int     `gorm:"column:id" json:"id"`
		Name          string  `gorm:"column:name" json:"name"`
		Fingerprint   string  `gorm:"column:fingerprint" json:"fingerprint"`
		Status        string  `gorm:"column:status" json:"status"`
		CertPem       string  `gorm:"column:cert_pem" json:"certificatePem"`
		KeyPem        string  `gorm:"column:key_pem" json:"privateKeyPem"`
		ChainPem      string  `gorm:"column:chain_pem" json:"chainPem"`
		Issuer        string  `gorm:"column:issuer" json:"issuer"`
		IssueAt       string  `gorm:"column:issue_at" json:"issueAt"`
		ExpireAt      string  `gorm:"column:expire_at" json:"expireAt"`
		Source        string  `gorm:"column:source" json:"source"`
		RenewMode     string  `gorm:"column:renew_mode" json:"renewMode"`
		AcmeAccountID int     `gorm:"column:acme_account_id" json:"acmeAccountId"`
		Renewing      bool    `gorm:"column:renewing" json:"renewing"`
		LastError     *string `gorm:"column:last_error" json:"lastError"`
		CreatedAt     string  `gorm:"column:created_at" json:"createdAt"`
		UpdatedAt     string  `gorm:"column:updated_at" json:"updatedAt"`
	}

	var cert CertificateDetail
	if err := h.db.Table("certificates").
		Where("id = ?", certID).
		First(&cert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("Certificate not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrInternalError("Failed to query certificate", err))
		return
	}

	// Get certificate domains
	type DomainRow struct {
		Domain     string `gorm:"column:domain" json:"domain"`
		IsWildcard bool   `gorm:"column:is_wildcard" json:"isWildcard"`
	}

	var domains []DomainRow
	if err := h.db.Table("certificate_domains").
		Select("domain, is_wildcard").
		Where("certificate_id = ?", certID).
		Order("domain ASC").
		Scan(&domains).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to query certificate domains", err))
		return
	}

	// Build response
	type CertificateResponse struct {
		CertificateDetail
		Domains []DomainRow `json:"domains"`
	}

	response := CertificateResponse{
		CertificateDetail: cert,
		Domains:           domains,
	}

	httpx.OK(c, response)
}

// UploadCertificate handles POST /api/v1/certificates/upload
func (h *Handler) UploadCertificate(c *gin.Context) {
	type UploadRequest struct {
		Provider       string   `json:"provider" binding:"required"`        // must be "manual"
		CertificatePem string   `json:"certificatePem" binding:"required"`
		PrivateKeyPem  string   `json:"privateKeyPem" binding:"required"`
		Domains        []string `json:"domains" binding:"required,min=1"` // ["a.com","*.a.com"]
	}

	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid request: "+err.Error()))
		return
	}

	// Validate provider must be "manual"
	if req.Provider != "manual" {
		httpx.FailErr(c, httpx.ErrParamInvalid("provider must be 'manual'"))
		return
	}

	// Parse certificate to extract metadata
	certInfo, err := h.certService.ParseCertificate(req.CertificatePem)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid certificate PEM: "+err.Error()))
		return
	}

	// Validate private key
	if err := h.certService.ValidatePrivateKey(req.PrivateKeyPem); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid private key PEM: "+err.Error()))
		return
	}

	// Check fingerprint uniqueness
	var existingID int
	if err := h.db.Table("certificates").
		Select("id").
		Where("fingerprint = ?", certInfo.Fingerprint).
		Scan(&existingID).Error; err == nil && existingID > 0 {
		httpx.FailErr(c, httpx.ErrAlreadyExists("Certificate with same fingerprint already exists"))
		return
	}

	// Begin transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Insert certificate
	type CertificateInsert struct {
		Name        string `gorm:"column:name"`
		Fingerprint string `gorm:"column:fingerprint"`
		Status      string `gorm:"column:status"`
		CertPem     string `gorm:"column:cert_pem"`
		KeyPem      string `gorm:"column:key_pem"`
		ChainPem    string `gorm:"column:chain_pem"`
		Issuer      string `gorm:"column:issuer"`
		IssueAt     string `gorm:"column:issue_at"`
		ExpireAt    string `gorm:"column:expire_at"`
		Source      string `gorm:"column:source"`
		RenewMode   string `gorm:"column:renew_mode"`
	}

	certInsert := CertificateInsert{
		Name:        certInfo.CommonName,
		Fingerprint: certInfo.Fingerprint,
		Status:      certInfo.Status,
		CertPem:     req.CertificatePem,
		KeyPem:      req.PrivateKeyPem,
		ChainPem:    "",
		Issuer:      certInfo.Issuer,
		IssueAt:     certInfo.IssueAt,
		ExpireAt:    certInfo.ExpireAt,
		Source:      "manual",
		RenewMode:   "manual",
	}

	if err := tx.Table("certificates").Create(&certInsert).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrInternalError("Failed to insert certificate", err))
		return
	}

	// Get inserted certificate ID
	var certID int
	if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&certID).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get certificate ID", err))
		return
	}

	// Insert certificate domains
	for _, domain := range req.Domains {
		isWildcard := len(domain) > 2 && domain[:2] == "*."
		domainInsert := map[string]interface{}{
			"certificate_id": certID,
			"domain":         domain,
			"is_wildcard":    isWildcard,
		}
		if err := tx.Table("certificate_domains").Create(domainInsert).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrInternalError("Failed to insert certificate domain", err))
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to commit transaction", err))
		return
	}

	httpx.OK(c, gin.H{
		"id":          certID,
		"fingerprint": certInfo.Fingerprint,
		"domains":     req.Domains,
	})
}
