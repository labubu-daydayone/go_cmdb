package acme

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_cmdb/internal/acme"
	"go_cmdb/internal/httpx"
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

	httpx.OKItems(c, requests, total, page, pageSize)
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
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// Get provider
	provider, err := h.service.GetProvider(req.ProviderName)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to get provider", err))
		return
	}

	if provider == nil {
		httpx.FailErr(c, httpx.ErrNotFound("provider not found"))
		return
	}

	// Check if account already exists
	existingAccount, err := h.service.GetAccount(provider.ID, req.Email)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to check existing account", err))
		return
	}

	if existingAccount != nil {
		httpx.FailErr(c, httpx.ErrStateConflict("account already exists for this provider and email"))
		return
	}

	// Validate EAB credentials for providers that require them
	if provider.RequiresEAB {
		if req.EabKid == "" || req.EabHmacKey == "" {
			httpx.FailErr(c, httpx.ErrParamInvalid("EAB credentials required for this provider"))
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
		httpx.FailErr(c, httpx.ErrInternalError("failed to create account", err))
		return
	}

	// Prepare response item
	item := map[string]interface{}{
		"id":         account.ID,
		"providerId": account.ProviderID,
		"email":      account.Email,
		"status":     account.Status,
	}
	if account.EabKid != "" {
		item["eabKid"] = account.EabKid
	}

	// Return as items array (T0-STD-01 compliance)
	httpx.OKItems(c, []interface{}{item}, 1, 1, 1)
}

// ListProviders returns all ACME providers
// GET /api/v1/acme/providers
func (h *Handler) ListProviders(c *gin.Context) {
	var providers []model.AcmeProvider
	if err := h.db.Find(&providers).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to query providers", err))
		return
	}

	type ProviderItem struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		DirectoryURL string `json:"directoryUrl"`
		RequiresEAB  bool   `json:"requiresEab"`
		Status       string `json:"status"`
		CreatedAt    string `json:"createdAt"`
		UpdatedAt    string `json:"updatedAt"`
	}

	items := make([]ProviderItem, 0, len(providers))
	for _, p := range providers {
		items = append(items, ProviderItem{
			ID:           int64(p.ID),
			Name:         p.Name,
			DirectoryURL: p.DirectoryURL,
			RequiresEAB:  p.RequiresEAB,
			Status:       p.Status,
			CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	httpx.OKItems(c, items, int64(len(items)), 1, len(items))
}

// ListAccounts returns ACME accounts with filters and pagination
// GET /api/v1/acme/accounts?page=&pageSize=&providerId=&status=&email=
func (h *Handler) ListAccounts(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	providerIDStr := c.Query("providerId")
	status := c.Query("status")
	email := c.Query("email")

	// Build query
	query := h.db.Table("acme_accounts a").
		Select(`
			a.id,
			a.provider_id,
			p.name as provider_name,
			a.email,
			a.status,
			a.registration_uri,
			a.eab_kid,
			a.eab_expires_at,
			a.last_error,
			a.created_at,
			a.updated_at
		`).
		Joins("LEFT JOIN acme_providers p ON p.id = a.provider_id")

	// Apply filters
	if providerIDStr != "" {
		if providerID, err := strconv.ParseInt(providerIDStr, 10, 64); err == nil {
			query = query.Where("a.provider_id = ?", providerID)
		}
	}
	if status != "" {
		query = query.Where("a.status = ?", status)
	}
	if email != "" {
		query = query.Where("a.email LIKE ?", "%"+email+"%")
	}

	// Count total
	var total int64
	countQuery := h.db.Table("acme_accounts a")
	if providerIDStr != "" {
		if providerID, err := strconv.ParseInt(providerIDStr, 10, 64); err == nil {
			countQuery = countQuery.Where("a.provider_id = ?", providerID)
		}
	}
	if status != "" {
		countQuery = countQuery.Where("a.status = ?", status)
	}
	if email != "" {
		countQuery = countQuery.Where("a.email LIKE ?", "%"+email+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to count accounts", err))
		return
	}

	// Query with pagination
	offset := (page - 1) * pageSize
	query = query.Order("a.id DESC").Limit(pageSize).Offset(offset)

	type QueryRow struct {
		ID              int64   `gorm:"column:id"`
		ProviderID      int64   `gorm:"column:provider_id"`
		ProviderName    string  `gorm:"column:provider_name"`
		Email           string  `gorm:"column:email"`
		Status          string  `gorm:"column:status"`
		RegistrationURI *string `gorm:"column:registration_uri"`
		EABKid          *string `gorm:"column:eab_kid"`
		EABExpiresAt    *string `gorm:"column:eab_expires_at"`
		LastError       *string `gorm:"column:last_error"`
		CreatedAt       string  `gorm:"column:created_at"`
		UpdatedAt       string  `gorm:"column:updated_at"`
	}

	var rows []QueryRow
	if err := query.Scan(&rows).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to query accounts", err))
		return
	}

	type AccountItem struct {
		ID              int64   `json:"id"`
		ProviderID      int64   `json:"providerId"`
		ProviderName    string  `json:"providerName"`
		Email           string  `json:"email"`
		Status          string  `json:"status"`
		RegistrationURI *string `json:"registrationUri"`
		EABKid          *string `json:"eabKid"`
		EABExpiresAt    *string `json:"eabExpiresAt"`
		LastError       *string `json:"lastError"`
		CreatedAt       string  `json:"createdAt"`
		UpdatedAt       string  `json:"updatedAt"`
	}

	items := make([]AccountItem, 0, len(rows))
	for _, row := range rows {
		// Mask EAB Kid (show first 8 chars + ***)
		var maskedEABKid *string
		if row.EABKid != nil && *row.EABKid != "" {
			kid := *row.EABKid
			if len(kid) > 8 {
				masked := kid[:8] + "***"
				maskedEABKid = &masked
			} else {
				maskedEABKid = row.EABKid
			}
		}

		items = append(items, AccountItem{
			ID:              row.ID,
			ProviderID:      row.ProviderID,
			ProviderName:    row.ProviderName,
			Email:           row.Email,
			Status:          row.Status,
			RegistrationURI: row.RegistrationURI,
			EABKid:          maskedEABKid,
			EABExpiresAt:    row.EABExpiresAt,
			LastError:       row.LastError,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}

	httpx.OKItems(c, items, total, page, pageSize)
}

// GetAccount returns a single ACME account by ID
// GET /api/v1/acme/accounts/:id
func (h *Handler) GetAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid( "must be a valid integer"))
		return
	}

	type QueryRow struct {
		ID              int64   `gorm:"column:id"`
		ProviderID      int64   `gorm:"column:provider_id"`
		ProviderName    string  `gorm:"column:provider_name"`
		Email           string  `gorm:"column:email"`
		Status          string  `gorm:"column:status"`
		RegistrationURI *string `gorm:"column:registration_uri"`
		EABKid          *string `gorm:"column:eab_kid"`
		EABExpiresAt    *string `gorm:"column:eab_expires_at"`
		LastError       *string `gorm:"column:last_error"`
		CreatedAt       string  `gorm:"column:created_at"`
		UpdatedAt       string  `gorm:"column:updated_at"`
	}

	var row QueryRow
	err = h.db.Table("acme_accounts a").
		Select(`
			a.id,
			a.provider_id,
			p.name as provider_name,
			a.email,
			a.status,
			a.registration_uri,
			a.eab_kid,
			a.eab_expires_at,
			a.last_error,
			a.created_at,
			a.updated_at
		`).
		Joins("LEFT JOIN acme_providers p ON p.id = a.provider_id").
		Where("a.id = ?", id).
		Scan(&row).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound( idStr))
			return
		}
		httpx.FailErr(c, httpx.ErrInternalError("failed to query account", err))
		return
	}

	// Mask EAB Kid
	var maskedEABKid *string
	if row.EABKid != nil && *row.EABKid != "" {
		kid := *row.EABKid
		if len(kid) > 8 {
			masked := kid[:8] + "***"
			maskedEABKid = &masked
		} else {
			maskedEABKid = row.EABKid
		}
	}

	result := gin.H{
		"id":              row.ID,
		"providerId":      row.ProviderID,
		"providerName":    row.ProviderName,
		"email":           row.Email,
		"status":          row.Status,
		"registrationUri": row.RegistrationURI,
		"eabKid":          maskedEABKid,
		"eabExpiresAt":    row.EABExpiresAt,
		"lastError":       row.LastError,
		"createdAt":       row.CreatedAt,
		"updatedAt":       row.UpdatedAt,
	}

	httpx.OK(c, result)
}

// EnableAccountRequest represents enable account request
type EnableAccountRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// EnableAccount enables an ACME account
// POST /api/v1/acme/accounts/enable
func (h *Handler) EnableAccount(c *gin.Context) {
	var req EnableAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid( "must be provided"))
		return
	}

	// Update status to active (idempotent)
	result := h.db.Model(&model.AcmeAccount{}).
		Where("id = ?", req.ID).
		Update("status", "active")

	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to enable account", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		httpx.FailErr(c, httpx.ErrNotFound("account not found"))
		return
	}

	// Return as items array (T0-STD-01 compliance)
	item := map[string]interface{}{"id": req.ID, "status": "active"}
	httpx.OKItems(c, []interface{}{item}, 1, 1, 1)
}

// DisableAccountRequest represents disable account request
type DisableAccountRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// DisableAccount disables an ACME account
// POST /api/v1/acme/accounts/disable
func (h *Handler) DisableAccount(c *gin.Context) {
	var req DisableAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("id must be provided"))
		return
	}

	// Check if account is default (禁止禁用default账号)
	var defaultCount int64
	if err := h.db.Table("acme_provider_defaults").
		Where("account_id = ?", req.ID).
		Count(&defaultCount).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to check default status", err))
		return
	}

	if defaultCount > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cannot disable default account, please set another account as default first"))
		return
	}

	// Update status to disabled (idempotent)
	result := h.db.Model(&model.AcmeAccount{}).
		Where("id = ?", req.ID).
		Update("status", "disabled")

	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to disable account", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		httpx.FailErr(c, httpx.ErrNotFound("account not found"))
		return
	}

	// Return as items array (T0-STD-01 compliance)
	item := map[string]interface{}{"id": req.ID, "status": "disabled"}
	httpx.OKItems(c, []interface{}{item}, 1, 1, 1)
}

// ListDefaults returns default ACME accounts for each provider
// GET /api/v1/acme/accounts/defaults
func (h *Handler) ListDefaults(c *gin.Context) {
	type QueryRow struct {
		ProviderID   int64  `gorm:"column:provider_id"`
		ProviderName string `gorm:"column:provider_name"`
		AccountID    int64  `gorm:"column:account_id"`
		AccountEmail string `gorm:"column:account_email"`
		UpdatedAt    string `gorm:"column:updated_at"`
	}

	var rows []QueryRow
	err := h.db.Table("acme_provider_defaults d").
		Select(`
			d.provider_id,
			p.name as provider_name,
			d.account_id,
			a.email as account_email,
			d.updated_at
		`).
		Joins("LEFT JOIN acme_providers p ON p.id = d.provider_id").
		Joins("LEFT JOIN acme_accounts a ON a.id = d.account_id").
		Scan(&rows).Error

	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to query defaults", err))
		return
	}

	type DefaultItem struct {
		ProviderID   int64  `json:"providerId"`
		ProviderName string `json:"providerName"`
		AccountID    int64  `json:"accountId"`
		AccountEmail string `json:"accountEmail"`
		UpdatedAt    string `json:"updatedAt"`
	}

	items := make([]DefaultItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, DefaultItem{
			ProviderID:   row.ProviderID,
			ProviderName: row.ProviderName,
			AccountID:    row.AccountID,
			AccountEmail: row.AccountEmail,
			UpdatedAt:    row.UpdatedAt,
		})
	}

	httpx.OKItems(c, items, int64(len(items)), 1, len(items))
}

// SetDefaultRequest represents set default account request
type SetDefaultRequest struct {
	ProviderID int64 `json:"providerId" binding:"required"`
	AccountID  int64 `json:"accountId" binding:"required"`
}

// SetDefault sets the default ACME account for a provider
// POST /api/v1/acme/accounts/set-default
func (h *Handler) SetDefault(c *gin.Context) {
	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid( "must be provided"))
		return
	}

	// Validate provider exists and is active
	var provider model.AcmeProvider
	if err := h.db.First(&provider, req.ProviderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound( strconv.FormatInt(req.ProviderID, 10)))
			return
		}
		httpx.FailErr(c, httpx.ErrInternalError("failed to query provider", err))
		return
	}
	if provider.Status != "active" {
		httpx.FailErr(c, httpx.ErrStateConflict( "provider is not active"))
		return
	}

	// Validate account exists, matches provider, and is active
	var account model.AcmeAccount
	if err := h.db.First(&account, req.AccountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound( strconv.FormatInt(req.AccountID, 10)))
			return
		}
		httpx.FailErr(c, httpx.ErrInternalError("failed to query account", err))
		return
	}
	if int64(account.ProviderID) != req.ProviderID {
		httpx.FailErr(c, httpx.ErrStateConflict( "account does not belong to the specified provider"))
		return
	}
	if account.Status != "active" {
		httpx.FailErr(c, httpx.ErrStateConflict( "account is not active"))
		return
	}

	// Upsert default (replace if exists)
	var defaultRecord model.ACMEProviderDefault
	result := h.db.Where("provider_id = ?", req.ProviderID).First(&defaultRecord)

	if result.Error == gorm.ErrRecordNotFound {
		// Insert new
		defaultRecord = model.ACMEProviderDefault{
			ProviderID: req.ProviderID,
			AccountID:  req.AccountID,
		}
		if err := h.db.Create(&defaultRecord).Error; err != nil {
			httpx.FailErr(c, httpx.ErrInternalError("failed to create default", err))
			return
		}
	} else if result.Error != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to query default", result.Error))
		return
	} else {
		// Update existing
		if err := h.db.Model(&defaultRecord).Update("account_id", req.AccountID).Error; err != nil {
			httpx.FailErr(c, httpx.ErrInternalError("failed to update default", err))
			return
		}
	}

	// Return as items array (T0-STD-01 compliance)
	item := map[string]interface{}{
		"providerId": req.ProviderID,
		"accountId":  req.AccountID,
	}
	httpx.OKItems(c, []interface{}{item}, 1, 1, 1)
}

// DeleteAccountRequest represents delete account request
type DeleteAccountRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// DeleteAccount deletes an ACME account (with constraints)
// POST /api/v1/acme/accounts/delete
func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("id must be provided"))
		return
	}

	// Check if account is default (禁止删除default账号)
	var defaultCount int64
	if err := h.db.Table("acme_provider_defaults").
		Where("account_id = ?", req.ID).
		Count(&defaultCount).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to check default status", err))
		return
	}

	if defaultCount > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cannot delete default account, please set another account as default first"))
		return
	}

	// Check if account is referenced by certificate_requests
	var certRequestCount int64
	if err := h.db.Table("certificate_requests").
		Where("acme_account_id = ?", req.ID).
		Count(&certRequestCount).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to check certificate request references", err))
		return
	}

	if certRequestCount > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cannot delete account, it is referenced by certificate requests"))
		return
	}

	// Check if account is referenced by certificates
	var certCount int64
	if err := h.db.Table("certificates").
		Where("acme_account_id = ?", req.ID).
		Count(&certCount).Error; err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to check certificate references", err))
		return
	}

	if certCount > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cannot delete account, it is referenced by certificates"))
		return
	}

	// Delete account
	result := h.db.Delete(&model.AcmeAccount{}, req.ID)

	if result.Error != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to delete account", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		httpx.FailErr(c, httpx.ErrNotFound("account not found"))
		return
	}

	// Return as items array (T0-STD-01 compliance)
	item := map[string]interface{}{"id": req.ID, "deleted": true}
	httpx.OKItems(c, []interface{}{item}, 1, 1, 1)
}
