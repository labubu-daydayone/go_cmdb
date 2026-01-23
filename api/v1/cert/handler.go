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
