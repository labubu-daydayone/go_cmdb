package acme

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_cmdb/internal/acme"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles ACME certificate requests
type Handler struct {
	db      *gorm.DB
	service *acme.Service
}

// NewHandler creates a new ACME handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:      db,
		service: acme.NewService(db),
	}
}

// RequestCertificateRequest represents a certificate request
type RequestCertificateRequest struct {
	AccountID int      `json:"accountId" binding:"required"`
	Domains   []string `json:"domains" binding:"required,min=1"`
}

// RequestCertificate creates a new certificate request
// POST /api/v1/acme/certificate/request
func (h *Handler) RequestCertificate(c *gin.Context) {
	var req RequestCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	// Validate account exists
	var account model.AcmeAccount
	if err := h.db.First(&account, req.AccountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    1,
				"message": "Account not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	// Serialize domains to JSON
	domainsJSON, err := json.Marshal(req.Domains)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to serialize domains",
		})
		return
	}

	// Create certificate request
	certRequest := &model.CertificateRequest{
		AccountID:       req.AccountID,
		Domains:         string(domainsJSON),
		Status:          model.CertificateRequestStatusPending,
		Attempts:        0,
		PollMaxAttempts: 10,
	}

	if err := h.service.CreateRequest(certRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to create certificate request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    certRequest,
	})
}

// RetryRequest retries a failed certificate request
// POST /api/v1/acme/certificate/retry
func (h *Handler) RetryRequest(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	// Reset retry
	if err := h.service.ResetRetry(req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to reset retry",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// ListRequests lists certificate requests
// GET /api/v1/acme/certificate/requests
func (h *Handler) ListRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	accountIDStr := c.Query("accountId")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}
	if accountIDStr != "" {
		accountID, err := strconv.Atoi(accountIDStr)
		if err == nil {
			filters["accountId"] = accountID
		}
	}

	requests, total, err := h.service.ListRequests(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     requests,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// GetRequest gets a single certificate request
// GET /api/v1/acme/certificate/requests/:id
func (h *Handler) GetRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "Invalid request ID",
		})
		return
	}

	request, err := h.service.GetRequest(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	if request == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "Request not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    request,
	})
}

// CreateAccountRequest represents an ACME account creation request
type CreateAccountRequest struct {
	ProviderName string  `json:"providerName" binding:"required"` // letsencrypt or google
	Email        string  `json:"email" binding:"required,email"`
	EabKid       string  `json:"eabKid"` // Required for Google Public CA
	EabHmacKey   string  `json:"eabHmacKey"` // Required for Google Public CA
}

// CreateAccount creates a new ACME account
// POST /api/v1/acme/account/create
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	// Get provider
	provider, err := h.service.GetProvider(req.ProviderName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to get provider",
		})
		return
	}

	if provider == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "Provider not found",
		})
		return
	}

	// Check if account already exists
	existingAccount, err := h.service.GetAccount(provider.ID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to check existing account",
		})
		return
	}

	if existingAccount != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "Account already exists",
			"data":    existingAccount,
		})
		return
	}

	// Validate EAB credentials for providers that require them
	if provider.RequiresEAB {
		if req.EabKid == "" || req.EabHmacKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1,
				"message": "EAB credentials required for this provider",
			})
			return
		}
	}

	// Create account
	account := &model.AcmeAccount{
		ProviderID:  provider.ID,
		Email:       req.Email,
		EabKid:      req.EabKid,
		EabHmacKey:  req.EabHmacKey,
		Status:      model.AcmeAccountStatusPending,
	}

	if err := h.service.CreateAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to create account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    account,
	})
}
