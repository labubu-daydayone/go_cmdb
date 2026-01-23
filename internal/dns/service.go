package dns

import (
	"fmt"
	"math"
	"time"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service provides DNS record management operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new DNS service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// GetDB returns the database instance
func (s *Service) GetDB() *gorm.DB {
	return s.db
}

// GetPendingRecords retrieves DNS records that need to be processed
// Filters:
// - status in ('pending', 'error')
// - next_retry_at is null or <= now
// - desired_state = 'present'
// - domain_dns_providers.provider = 'cloudflare'
// - domain_dns_providers.status = 'active'
// - domains.status = 'active'
func (s *Service) GetPendingRecords(limit int) ([]model.DomainDNSRecord, error) {
	var records []model.DomainDNSRecord

	err := s.db.
		Joins("JOIN domains ON domains.id = domain_dns_records.domain_id").
		Joins("JOIN domain_dns_providers ON domain_dns_providers.domain_id = domains.id").
		Where("domain_dns_records.status IN ?", []string{string(model.DNSRecordStatusPending), string(model.DNSRecordStatusError)}).
		Where("(domain_dns_records.next_retry_at IS NULL OR domain_dns_records.next_retry_at <= ?)", time.Now()).
		Where("domain_dns_records.desired_state = ?", model.DNSRecordDesiredStatePresent).
		Where("domain_dns_providers.provider = ?", "cloudflare").
		Where("domain_dns_providers.status = ?", "active").
		Where("domains.status = ?", "active").
		Limit(limit).
		Find(&records).Error

	return records, err
}

// GetDeletionRecords retrieves DNS records that need to be deleted
// Filters:
// - desired_state = 'absent'
// - provider_record_id is not null (has been synced to Cloudflare)
func (s *Service) GetDeletionRecords(limit int) ([]model.DomainDNSRecord, error) {
	var records []model.DomainDNSRecord

	err := s.db.
		Where("desired_state = ?", model.DNSRecordDesiredStateAbsent).
		Where("provider_record_id IS NOT NULL AND provider_record_id != ''").
		Limit(limit).
		Find(&records).Error

	return records, err
}

// MarkAsRunning marks a DNS record as running (being processed by worker)
// Uses optimistic locking to prevent concurrent processing
func (s *Service) MarkAsRunning(recordID int) error {
	result := s.db.Model(&model.DomainDNSRecord{}).
		Where("id = ?", recordID).
		Where("status IN ?", []string{string(model.DNSRecordStatusPending), string(model.DNSRecordStatusError)}).
		Update("status", model.DNSRecordStatusRunning)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record %d is already being processed or not in pending/error state", recordID)
	}

	return nil
}

// MarkAsActive marks a DNS record as active (successfully synced)
func (s *Service) MarkAsActive(recordID int, providerRecordID string) error {
	updates := map[string]interface{}{
		"status":             model.DNSRecordStatusActive,
		"provider_record_id": providerRecordID,
		"last_error":         nil,
		// Keep retry_count for audit purposes (don't reset to 0)
	}

	return s.db.Model(&model.DomainDNSRecord{}).
		Where("id = ?", recordID).
		Updates(updates).Error
}

// MarkAsError marks a DNS record as error (sync failed)
func (s *Service) MarkAsError(recordID int, errorMsg string) error {
	var record model.DomainDNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return err
	}

	// Increment retry count
	retryCount := record.RetryCount + 1

	// Calculate next retry time using backoff strategy
	// backoff = min(2^retry_count * 30s, 30m)
	// If retry_count >= 10, stop automatic retry (next_retry_at = null)
	var nextRetryAt *time.Time
	if retryCount < 10 {
		backoffSeconds := math.Min(math.Pow(2, float64(retryCount))*30, 1800) // max 30 minutes
		nextRetry := time.Now().Add(time.Duration(backoffSeconds) * time.Second)
		nextRetryAt = &nextRetry
	}

	// Truncate error message to 255 characters (database field limit)
	if len(errorMsg) > 255 {
		errorMsg = errorMsg[:252] + "..."
	}

	updates := map[string]interface{}{
		"status":        model.DNSRecordStatusError,
		"last_error":    errorMsg,
		"retry_count":   retryCount,
		"next_retry_at": nextRetryAt,
	}

	return s.db.Model(&model.DomainDNSRecord{}).
		Where("id = ?", recordID).
		Updates(updates).Error
}

// DeleteRecord hard deletes a DNS record from the database
func (s *Service) DeleteRecord(recordID int) error {
	return s.db.Delete(&model.DomainDNSRecord{}, recordID).Error
}

// ResetRetry resets retry state for a DNS record (for manual retry)
func (s *Service) ResetRetry(recordID int) error {
	now := time.Now()
	updates := map[string]interface{}{
		"next_retry_at": &now,
		// Keep retry_count and last_error for audit purposes
	}

	return s.db.Model(&model.DomainDNSRecord{}).
		Where("id = ?", recordID).
		Where("status = ?", model.DNSRecordStatusError).
		Updates(updates).Error
}

// GetDomainProvider retrieves the DNS provider for a domain
func (s *Service) GetDomainProvider(domainID int) (*model.DomainDNSProvider, error) {
	var provider model.DomainDNSProvider
	err := s.db.Where("domain_id = ? AND status = ?", domainID, "active").First(&provider).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

// GetDomain retrieves a domain by ID
func (s *Service) GetDomain(domainID int) (*model.Domain, error) {
	var domain model.Domain
	err := s.db.First(&domain, domainID).Error
	if err != nil {
		return nil, err
	}
	return &domain, nil
}
