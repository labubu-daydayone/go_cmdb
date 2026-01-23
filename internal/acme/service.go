package acme

import (
	"fmt"
	"log"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service provides ACME-related database operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new ACME service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// GetDB returns the database instance
func (s *Service) GetDB() *gorm.DB {
	return s.db
}

// GetPendingRequests returns certificate requests that need processing
func (s *Service) GetPendingRequests(limit int) ([]model.CertificateRequest, error) {
	var requests []model.CertificateRequest
	err := s.db.
		Where("status IN (?, ?)", model.CertificateRequestStatusPending, model.CertificateRequestStatusRunning).
		Where("attempts < poll_max_attempts").
		Order("created_at ASC").
		Limit(limit).
		Find(&requests).Error
	return requests, err
}

// MarkAsRunning marks a certificate request as running (optimistic lock)
func (s *Service) MarkAsRunning(requestID int) error {
	result := s.db.
		Model(&model.CertificateRequest{}).
		Where("id = ? AND status IN (?, ?)", requestID, model.CertificateRequestStatusPending, model.CertificateRequestStatusRunning).
		Update("status", model.CertificateRequestStatusRunning)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("request %d already processed by another worker", requestID)
	}

	return nil
}

// MarkAsSuccess marks a certificate request as success
func (s *Service) MarkAsSuccess(requestID int, certificateID int) error {
	return s.db.
		Model(&model.CertificateRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":                model.CertificateRequestStatusSuccess,
			"result_certificate_id": certificateID,
			"last_error":            "",
		}).Error
}

// MarkAsFailed marks a certificate request as failed
func (s *Service) MarkAsFailed(requestID int, lastError string) error {
	var request model.CertificateRequest
	if err := s.db.First(&request, requestID).Error; err != nil {
		return err
	}

	attempts := request.Attempts + 1
	status := model.CertificateRequestStatusPending

	// If max attempts reached, mark as failed and cleanup challenge
	if attempts >= request.PollMaxAttempts {
		status = model.CertificateRequestStatusFailed
		
		// Cleanup challenge TXT records
		if err := s.CleanupFailedChallenge(requestID); err != nil {
			log.Printf("[ACME Service] Failed to cleanup challenge for request %d: %v\n", requestID, err)
			// Don't fail the whole operation, just log
		}
	}

	// Truncate error to 500 characters
	if len(lastError) > 500 {
		lastError = lastError[:500]
	}

	return s.db.
		Model(&model.CertificateRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":     status,
			"attempts":   attempts,
			"last_error": lastError,
		}).Error
}

// ResetRetry resets a failed request to pending for manual retry
// Note: attempts and last_error are preserved for audit trail
func (s *Service) ResetRetry(requestID int) error {
	return s.db.
		Model(&model.CertificateRequest{}).
		Where("id = ? AND status = ?", requestID, model.CertificateRequestStatusFailed).
		Updates(map[string]interface{}{
			"status":     model.CertificateRequestStatusPending,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

// GetAccount returns an ACME account by provider and email
func (s *Service) GetAccount(providerID int, email string) (*model.AcmeAccount, error) {
	var account model.AcmeAccount
	err := s.db.
		Where("provider_id = ? AND email = ?", providerID, email).
		First(&account).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &account, err
}

// CreateAccount creates a new ACME account
func (s *Service) CreateAccount(account *model.AcmeAccount) error {
	return s.db.Create(account).Error
}

// GetProvider returns an ACME provider by name
func (s *Service) GetProvider(name string) (*model.AcmeProvider, error) {
	var provider model.AcmeProvider
	err := s.db.Where("name = ?", name).First(&provider).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &provider, err
}

// CreateProvider creates a new ACME provider
func (s *Service) CreateProvider(provider *model.AcmeProvider) error {
	return s.db.Create(provider).Error
}

// GetRequest returns a certificate request by ID
func (s *Service) GetRequest(requestID int) (*model.CertificateRequest, error) {
	var request model.CertificateRequest
	err := s.db.First(&request, requestID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &request, err
}

// CreateRequest creates a new certificate request
func (s *Service) CreateRequest(request *model.CertificateRequest) error {
	return s.db.Create(request).Error
}

// ListRequests returns a paginated list of certificate requests
func (s *Service) ListRequests(page, pageSize int, filters map[string]interface{}) ([]model.CertificateRequest, int64, error) {
	var requests []model.CertificateRequest
	var total int64

	query := s.db.Model(&model.CertificateRequest{})

	// Apply filters
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if accountID, ok := filters["accountId"]; ok {
		query = query.Where("account_id = ?", accountID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// CreateCertificate creates a new certificate
func (s *Service) CreateCertificate(certificate *model.Certificate) error {
	return s.db.Create(certificate).Error
}

// CreateCertificateDomains creates certificate domains (SAN)
func (s *Service) CreateCertificateDomains(domains []model.CertificateDomain) error {
	if len(domains) == 0 {
		return nil
	}
	return s.db.Create(&domains).Error
}

// CreateCertificateBinding creates a certificate binding
func (s *Service) CreateCertificateBinding(binding *model.CertificateBinding) error {
	return s.db.Create(binding).Error
}

// GetCertificateBindingsByRequest returns certificate bindings for a request
// This is used to find websites that should be updated after certificate issuance
func (s *Service) GetCertificateBindingsByRequest(requestID int) ([]model.CertificateBinding, error) {
	var bindings []model.CertificateBinding
	
	// Find bindings where the certificate was created from this request
	err := s.db.
		Joins("JOIN certificates ON certificates.id = certificate_bindings.certificate_id").
		Joins("JOIN certificate_requests ON certificate_requests.result_certificate_id = certificates.id").
		Where("certificate_requests.id = ?", requestID).
		Find(&bindings).Error
	
	return bindings, err
}

// ActivateCertificateBinding activates a certificate binding
func (s *Service) ActivateCertificateBinding(bindingID int) error {
	return s.db.
		Model(&model.CertificateBinding{}).
		Where("id = ?", bindingID).
		Update("status", model.CertificateBindingStatusActive).Error
}

// OnCertificateIssued handles post-issuance actions: update bindings and trigger config apply
func (s *Service) OnCertificateIssued(requestID int, certID int) error {
	// Step 1: Find certificate bindings for this request
	var bindings []model.CertificateBinding
	if err := s.db.Where("certificate_request_id = ?", requestID).Find(&bindings).Error; err != nil {
		return fmt.Errorf("failed to query certificate_bindings: %w", err)
	}

	if len(bindings) == 0 {
		// No bindings, skip auto-apply
		return nil
	}

	// Step 2: Update website_https.certificate_id and activate bindings
	for _, binding := range bindings {
		// Update website_https.certificate_id
		if err := s.db.Exec(`
			UPDATE website_https 
			SET certificate_id = ?, updated_at = NOW() 
			WHERE website_id = ? AND enabled = TRUE AND cert_mode = ?
		`, certID, binding.WebsiteID, model.CertModeACME).Error; err != nil {
			return fmt.Errorf("failed to update website_https.certificate_id: %w", err)
		}

		// Activate certificate_binding
		if err := s.db.Model(&binding).Updates(map[string]interface{}{
			"status":     model.CertificateBindingStatusActive,
			"updated_at": gorm.Expr("NOW()"),
		}).Error; err != nil {
			return fmt.Errorf("failed to activate certificate_binding: %w", err)
		}
	}

	// Step 3: Trigger config apply (idempotency check via reason)
	reason := fmt.Sprintf("acme-issued:%d", certID)

	// Check if already triggered
	var existingCount int64
	if err := s.db.Model(&model.ConfigVersion{}).
		Where("reason = ?", reason).
		Count(&existingCount).Error; err != nil {
		return fmt.Errorf("failed to check existing config_versions: %w", err)
	}

	if existingCount > 0 {
		// Already triggered, skip
		return nil
	}

	// Step 4: Create config_version (using DB auto-increment for version)
	configVersion := &model.ConfigVersion{
		NodeID:  0, // 0 means all nodes (full apply)
		Version: 0, // Will be set to ID after insert
		Payload: "", // Will be generated by aggregator
		Status:  model.ConfigVersionStatusPending,
		Reason:  reason,
	}

	if err := s.db.Create(configVersion).Error; err != nil {
		return fmt.Errorf("failed to create config_version: %w", err)
	}

	// Update version = id
	if err := s.db.Model(configVersion).Update("version", configVersion.ID).Error; err != nil {
		return fmt.Errorf("failed to update config_version.version: %w", err)
	}

	// Step 5: Generate payload and update config_version
	// Note: This requires configgen.Aggregator, which we'll import
	// For now, we'll create a placeholder and let the apply API handle payload generation
	// Alternative: Import configgen here and generate payload immediately

	// Step 6: Create agent_tasks for all enabled nodes
	var nodes []model.Node
	if err := s.db.Where("enabled = ?", true).Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to query enabled nodes: %w", err)
	}

	for _, node := range nodes {
		task := &model.AgentTask{
			NodeID:  node.ID,
			Type:    "apply_config",
			Payload: fmt.Sprintf(`{"version": %d}`, configVersion.Version),
			Status:  model.TaskStatusPending,
		}

		if err := s.db.Create(task).Error; err != nil {
			return fmt.Errorf("failed to create agent_task: %w", err)
		}
	}

	return nil
}

// CleanupFailedChallenge cleans up challenge TXT records for failed requests
func (s *Service) CleanupFailedChallenge(requestID int) error {
	// Update all challenge TXT records for this request to desired_state=absent
	// DNS Worker will handle cloud deletion and eventual hard delete
	result := s.db.
		Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id = ? AND desired_state = ?", 
			model.DNSRecordOwnerACMEChallenge, 
			requestID, 
			model.DNSRecordDesiredStatePresent).
		Update("desired_state", model.DNSRecordDesiredStateAbsent)

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup challenge records: %w", result.Error)
	}

	log.Printf("[ACME Service] Cleaned up %d challenge records for request %d\n", result.RowsAffected, requestID)
	return nil
}
