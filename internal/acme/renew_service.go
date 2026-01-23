package acme

import (
	"encoding/json"
	"fmt"
	"go_cmdb/internal/model"
	"time"

	"gorm.io/gorm"
)

// RenewService handles certificate renewal operations
type RenewService struct {
	db *gorm.DB
}

// NewRenewService creates a new RenewService
func NewRenewService(db *gorm.DB) *RenewService {
	return &RenewService{db: db}
}

// GetRenewCandidates returns certificates that need renewal
// Criteria:
// - status = valid
// - expire_at <= now + renewBeforeDays
// - source = acme
// - renew_mode = auto
// - acme_account_id is not null
// - renewing = false (not already renewing)
func (s *RenewService) GetRenewCandidates(renewBeforeDays int, limit int) ([]model.Certificate, error) {
	var certificates []model.Certificate
	renewThreshold := time.Now().AddDate(0, 0, renewBeforeDays)

	err := s.db.Where("status = ?", model.CertificateStatusValid).
		Where("expire_at <= ?", renewThreshold).
		Where("source = ?", model.CertificateSourceAcme).
		Where("renew_mode = ?", model.CertificateRenewModeAuto).
		Where("acme_account_id IS NOT NULL AND acme_account_id > 0").
		Where("renewing = ?", false).
		Limit(limit).
		Find(&certificates).Error

	return certificates, err
}

// MarkAsRenewing marks a certificate as renewing (sets renewing flag)
// Uses optimistic locking to prevent concurrent renewal
func (s *RenewService) MarkAsRenewing(certID int) error {
	result := s.db.Model(&model.Certificate{}).
		Where("id = ?", certID).
		Where("renewing = ?", false). // Optimistic lock
		Update("renewing", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("certificate %d is already renewing or not found", certID)
	}

	return nil
}

// ClearRenewing clears the renewing flag
func (s *RenewService) ClearRenewing(certID int) error {
	return s.db.Model(&model.Certificate{}).
		Where("id = ?", certID).
		Update("renewing", false).Error
}

// CreateRenewRequest creates a certificate_requests record for renewal
func (s *RenewService) CreateRenewRequest(certID int, accountID int, domains []string) (*model.CertificateRequest, error) {
	// Marshal domains to JSON
	domainsJSON, err := json.Marshal(domains)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal domains: %w", err)
	}

	renewCertID := certID
	request := &model.CertificateRequest{
		AccountID:       accountID,
		Domains:         string(domainsJSON),
		Status:          model.CertificateRequestStatusPending,
		Attempts:        0,
		PollMaxAttempts: 10,
		RenewCertID:     &renewCertID,
	}

	if err := s.db.Create(request).Error; err != nil {
		return nil, err
	}

	return request, nil
}

// GetCertificateDomains returns domains for a certificate
func (s *RenewService) GetCertificateDomains(certID int) ([]string, error) {
	var domains []model.CertificateDomain
	if err := s.db.Where("certificate_id = ?", certID).Find(&domains).Error; err != nil {
		return nil, err
	}

	result := make([]string, len(domains))
	for i, d := range domains {
		result[i] = d.Domain
	}

	return result, nil
}

// GetCertificate returns a certificate by ID
func (s *RenewService) GetCertificate(certID int) (*model.Certificate, error) {
	var cert model.Certificate
	if err := s.db.First(&cert, certID).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

// UpdateRenewMode updates the renew_mode of a certificate
func (s *RenewService) UpdateRenewMode(certID int, renewMode string) error {
	return s.db.Model(&model.Certificate{}).
		Where("id = ?", certID).
		Update("renew_mode", renewMode).Error
}

// ListRenewCandidates returns paginated list of renewal candidates
func (s *RenewService) ListRenewCandidates(renewBeforeDays int, status string, page int, pageSize int) ([]model.Certificate, int64, error) {
	var certificates []model.Certificate
	var total int64

	renewThreshold := time.Now().AddDate(0, 0, renewBeforeDays)

	query := s.db.Model(&model.Certificate{}).
		Where("expire_at <= ?", renewThreshold).
		Where("source = ?", model.CertificateSourceAcme).
		Where("renew_mode = ?", model.CertificateRenewModeAuto).
		Where("acme_account_id IS NOT NULL AND acme_account_id > 0")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&certificates).Error; err != nil {
		return nil, 0, err
	}

	return certificates, total, nil
}
