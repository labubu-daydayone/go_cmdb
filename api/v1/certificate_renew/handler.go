package certificate_renew

import (
	"fmt"
	"strconv"
	"go_cmdb/internal/acme"
	"go_cmdb/internal/httpx"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles certificate renewal API requests
type Handler struct {
	db           *gorm.DB
	renewService *acme.RenewService
}

// NewHandler creates a new handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:           db,
		renewService: acme.NewRenewService(db),
	}
}

// GetRenewalCandidatesRequest represents the request to get renewal candidates
type GetRenewalCandidatesRequest struct {
	RenewBeforeDays int    `form:"renewBeforeDays"` // Default: 30
	Status          string `form:"status"`          // Optional filter by status
	Page            int    `form:"page"`            // Default: 1
	PageSize        int    `form:"pageSize"`        // Default: 20
}

// CertificateInfo represents certificate information
type CertificateInfo struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	Domains       []string `json:"domains"`
	ExpireAt      string   `json:"expireAt"`
	IssueAt       string   `json:"issueAt"`
	Source        string   `json:"source"`
	RenewMode     string   `json:"renewMode"`
	AcmeAccountID int      `json:"acmeAccountId"`
	Renewing      bool     `json:"renewing"`
	LastError     string   `json:"lastError"`
}

// TriggerRenewalRequest represents the request to trigger renewal
type TriggerRenewalRequest struct {
	CertificateID int `json:"certificateId" binding:"required"`
}

// DisableAutoRenewRequest represents the request to disable auto-renewal
type DisableAutoRenewRequest struct {
	CertificateID int `json:"certificateId" binding:"required"`
}

// GetRenewalCandidates handles GET /api/v1/certificates/renewal/candidates
func (h *Handler) GetRenewalCandidates(c *gin.Context) {
	// Parse query parameters with defaults
	renewBeforeDays := 30
	if val := c.Query("renewBeforeDays"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			renewBeforeDays = parsed
		}
	}

	status := c.Query("status")

	page := 1
	if val := c.Query("page"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if val := c.Query("pageSize"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	// Get renewal candidates
	certificates, total, err := h.renewService.ListRenewCandidates(renewBeforeDays, status, page, pageSize)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get renewal candidates", err))
		return
	}

	// Build response
	certInfos := make([]CertificateInfo, len(certificates))
	for i, cert := range certificates {
		// Get domains
		domains, err := h.renewService.GetCertificateDomains(cert.ID)
		if err != nil {
			domains = []string{}
		}

		var acmeAccountID int
		if cert.AcmeAccountID != nil {
			acmeAccountID = *cert.AcmeAccountID
		}
		var lastError string
		if cert.LastError != nil {
			lastError = *cert.LastError
		}
		certInfos[i] = CertificateInfo{
			ID:            cert.ID,
			Name:          fmt.Sprintf("Certificate #%d", cert.ID),
			Status:        cert.Status,
			Domains:       domains,
			ExpireAt:      cert.ExpireAt.Format("2006-01-02 15:04:05"),
			IssueAt:       cert.IssueAt.Format("2006-01-02 15:04:05"),
			Source:        cert.Source,
			RenewMode:     cert.RenewMode,
			AcmeAccountID: acmeAccountID,
			Renewing:      false,
			LastError:     lastError,
		}
	}

	httpx.OK(c, gin.H{
		"certificates": certInfos,
		"total":        total,
		"page":         page,
		"pageSize":     pageSize,
	})
}

// TriggerRenewal handles POST /api/v1/certificates/renewal/trigger
func (h *Handler) TriggerRenewal(c *gin.Context) {
	var req TriggerRenewalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid request: "+err.Error()))
		return
	}

	// Get certificate
	cert, err := h.renewService.GetCertificate(req.CertificateID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrNotFound("Certificate not found"))
		return
	}

	// Validate certificate can be renewed
	if cert.Source != "acme" {
		httpx.FailErr(c, httpx.ErrParamInvalid("Only ACME certificates can be renewed"))
		return
	}

	if cert.AcmeAccountID == nil || *cert.AcmeAccountID == 0 {
		httpx.FailErr(c, httpx.ErrParamInvalid("Certificate has no acme_account_id"))
		return
	}

	// Mark as renewing
	if err := h.renewService.MarkAsRenewing(req.CertificateID); err != nil {
		httpx.FailErr(c, httpx.ErrStateConflict("Certificate is already renewing"))
		return
	}

	// Get domains
	domains, err := h.renewService.GetCertificateDomains(req.CertificateID)
	if err != nil {
		h.renewService.ClearRenewing(req.CertificateID)
		httpx.FailErr(c, httpx.ErrInternalError("Failed to get domains", err))
		return
	}

	if len(domains) == 0 {
		h.renewService.ClearRenewing(req.CertificateID)
		httpx.FailErr(c, httpx.ErrParamInvalid("Certificate has no domains"))
		return
	}

	// Create renewal request
	request, err := h.renewService.CreateRenewRequest(req.CertificateID, *cert.AcmeAccountID, domains)
	if err != nil {
		h.renewService.ClearRenewing(req.CertificateID)
		httpx.FailErr(c, httpx.ErrInternalError("Failed to create renewal request", err))
		return
	}

	httpx.OK(c, gin.H{
		"requestId": request.ID,
	})
}

// DisableAutoRenew handles POST /api/v1/certificates/renewal/disable-auto
func (h *Handler) DisableAutoRenew(c *gin.Context) {
	var req DisableAutoRenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("Invalid request: "+err.Error()))
		return
	}

	// Update renew_mode to manual
	if err := h.renewService.UpdateRenewMode(req.CertificateID, "manual"); err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("Failed to disable auto-renewal", err))
		return
	}

	httpx.OK(c, gin.H{
		"message": "Auto-renewal disabled successfully",
	})
}
