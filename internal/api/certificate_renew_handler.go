package api

import (
	"encoding/json"
	"fmt"
	"go_cmdb/internal/acme"
	"log"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// CertificateRenewHandler handles certificate renewal API requests
type CertificateRenewHandler struct {
	db           *gorm.DB
	renewService *acme.RenewService
}

// NewCertificateRenewHandler creates a new handler
func NewCertificateRenewHandler(db *gorm.DB) *CertificateRenewHandler {
	return &CertificateRenewHandler{
		db:           db,
		renewService: acme.NewRenewService(db),
	}
}

// GetRenewalCandidatesRequest represents the request to get renewal candidates
type GetRenewalCandidatesRequest struct {
	RenewBeforeDays int    `json:"renewBeforeDays"` // Default: 30
	Status          string `json:"status"`          // Optional filter by status
	Page            int    `json:"page"`            // Default: 1
	PageSize        int    `json:"pageSize"`        // Default: 20
}

// GetRenewalCandidatesResponse represents the response
type GetRenewalCandidatesResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    GetRenewalCandidatesData `json:"data"`
}

// GetRenewalCandidatesData represents the data in response
type GetRenewalCandidatesData struct {
	Certificates []CertificateInfo `json:"certificates"`
	Total        int64             `json:"total"`
	Page         int               `json:"page"`
	PageSize     int               `json:"pageSize"`
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
	CertificateID int `json:"certificateId"`
}

// TriggerRenewalResponse represents the response
type TriggerRenewalResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    TriggerRenewalData   `json:"data"`
}

// TriggerRenewalData represents the data in response
type TriggerRenewalData struct {
	RequestID int `json:"requestId"`
}

// DisableAutoRenewRequest represents the request to disable auto-renewal
type DisableAutoRenewRequest struct {
	CertificateID int `json:"certificateId"`
}

// DisableAutoRenewResponse represents the response
type DisableAutoRenewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetRenewalCandidates handles GET /api/v1/certificates/renewal/candidates
func (h *CertificateRenewHandler) GetRenewalCandidates(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	renewBeforeDays := 30
	if val := r.URL.Query().Get("renewBeforeDays"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			renewBeforeDays = parsed
		}
	}

	status := r.URL.Query().Get("status")

	page := 1
	if val := r.URL.Query().Get("page"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if val := r.URL.Query().Get("pageSize"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	// Get renewal candidates
	certificates, total, err := h.renewService.ListRenewCandidates(renewBeforeDays, status, page, pageSize)
	if err != nil {
		log.Printf("[CertificateRenewHandler] Failed to get renewal candidates: %v\n", err)
		respondJSON(w, http.StatusInternalServerError, GetRenewalCandidatesResponse{
			Code:    500,
			Message: fmt.Sprintf("Failed to get renewal candidates: %v", err),
		})
		return
	}

	// Build response
	certInfos := make([]CertificateInfo, len(certificates))
	for i, cert := range certificates {
		// Get domains
		domains, err := h.renewService.GetCertificateDomains(cert.ID)
		if err != nil {
			log.Printf("[CertificateRenewHandler] Failed to get domains for certificate %d: %v\n", cert.ID, err)
			domains = []string{}
		}

		certInfos[i] = CertificateInfo{
			ID:            cert.ID,
			Name:          cert.Name,
			Status:        cert.Status,
			Domains:       domains,
			ExpireAt:      cert.ExpireAt.Format("2006-01-02 15:04:05"),
			IssueAt:       cert.IssueAt.Format("2006-01-02 15:04:05"),
			Source:        cert.Source,
			RenewMode:     cert.RenewMode,
			AcmeAccountID: cert.AcmeAccountID,
			Renewing:      cert.Renewing,
			LastError:     cert.LastError,
		}
	}

	respondJSON(w, http.StatusOK, GetRenewalCandidatesResponse{
		Code:    200,
		Message: "success",
		Data: GetRenewalCandidatesData{
			Certificates: certInfos,
			Total:        total,
			Page:         page,
			PageSize:     pageSize,
		},
	})
}

// TriggerRenewal handles POST /api/v1/certificates/renewal/trigger
func (h *CertificateRenewHandler) TriggerRenewal(w http.ResponseWriter, r *http.Request) {
	var req TriggerRenewalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, TriggerRenewalResponse{
			Code:    400,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Get certificate
	cert, err := h.renewService.GetCertificate(req.CertificateID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, TriggerRenewalResponse{
			Code:    404,
			Message: fmt.Sprintf("Certificate not found: %v", err),
		})
		return
	}

	// Validate certificate can be renewed
	if cert.Source != "acme" {
		respondJSON(w, http.StatusBadRequest, TriggerRenewalResponse{
			Code:    400,
			Message: "Only ACME certificates can be renewed",
		})
		return
	}

	if cert.AcmeAccountID == 0 {
		respondJSON(w, http.StatusBadRequest, TriggerRenewalResponse{
			Code:    400,
			Message: "Certificate has no acme_account_id",
		})
		return
	}

	// Mark as renewing
	if err := h.renewService.MarkAsRenewing(req.CertificateID); err != nil {
		respondJSON(w, http.StatusConflict, TriggerRenewalResponse{
			Code:    409,
			Message: fmt.Sprintf("Certificate is already renewing: %v", err),
		})
		return
	}

	// Get domains
	domains, err := h.renewService.GetCertificateDomains(req.CertificateID)
	if err != nil {
		h.renewService.ClearRenewing(req.CertificateID)
		respondJSON(w, http.StatusInternalServerError, TriggerRenewalResponse{
			Code:    500,
			Message: fmt.Sprintf("Failed to get domains: %v", err),
		})
		return
	}

	if len(domains) == 0 {
		h.renewService.ClearRenewing(req.CertificateID)
		respondJSON(w, http.StatusBadRequest, TriggerRenewalResponse{
			Code:    400,
			Message: "Certificate has no domains",
		})
		return
	}

	// Create renewal request
	request, err := h.renewService.CreateRenewRequest(req.CertificateID, cert.AcmeAccountID, domains)
	if err != nil {
		h.renewService.ClearRenewing(req.CertificateID)
		respondJSON(w, http.StatusInternalServerError, TriggerRenewalResponse{
			Code:    500,
			Message: fmt.Sprintf("Failed to create renewal request: %v", err),
		})
		return
	}

	log.Printf("[CertificateRenewHandler] Triggered renewal for certificate %d, request_id=%d\n", req.CertificateID, request.ID)

	respondJSON(w, http.StatusOK, TriggerRenewalResponse{
		Code:    200,
		Message: "Renewal triggered successfully",
		Data: TriggerRenewalData{
			RequestID: request.ID,
		},
	})
}

// DisableAutoRenew handles POST /api/v1/certificates/renewal/disable-auto
func (h *CertificateRenewHandler) DisableAutoRenew(w http.ResponseWriter, r *http.Request) {
	var req DisableAutoRenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, DisableAutoRenewResponse{
			Code:    400,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Update renew_mode to manual
	if err := h.renewService.UpdateRenewMode(req.CertificateID, "manual"); err != nil {
		respondJSON(w, http.StatusInternalServerError, DisableAutoRenewResponse{
			Code:    500,
			Message: fmt.Sprintf("Failed to disable auto-renewal: %v", err),
		})
		return
	}

	log.Printf("[CertificateRenewHandler] Disabled auto-renewal for certificate %d\n", req.CertificateID)

	respondJSON(w, http.StatusOK, DisableAutoRenewResponse{
		Code:    200,
		Message: "Auto-renewal disabled successfully",
	})
}

// respondJSON writes JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
