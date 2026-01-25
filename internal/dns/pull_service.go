package dns

import (
	"context"
	"fmt"
	"log"
	"time"

	"go_cmdb/internal/db"
	"go_cmdb/internal/dns/providers/cloudflare"
	"go_cmdb/internal/model"
)

// PullSyncResult represents the result of DNS records pull synchronization
type PullSyncResult struct {
	Fetched int `json:"fetched"` // Total records fetched from Cloudflare
	Created int `json:"created"` // New external records created
	Updated int `json:"updated"` // Existing records updated
	Deleted int `json:"deleted"` // Local records deleted (not in Cloudflare)
}

// PullSyncRecords pulls DNS records from Cloudflare and syncs to local database
// Core principle: Cloudflare is the source of truth
// Sync unit: provider_record_id (not name/value)
func PullSyncRecords(ctx context.Context, domainID int) (*PullSyncResult, error) {
	// 1. Validate domain exists
	var domain model.Domain
	if err := db.DB.Where("id = ?", domainID).First(&domain).Error; err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// 2. Get active provider binding
	var provider model.DomainDNSProvider
	if err := db.DB.Where("domain_id = ? AND status = ?", domainID, model.DNSProviderStatusActive).First(&provider).Error; err != nil {
		return nil, fmt.Errorf("active provider not found: %w", err)
	}

	// 3. Validate provider is cloudflare
	if provider.Provider != model.DNSProviderCloudflare {
		return nil, fmt.Errorf("only cloudflare provider is supported, got: %s", provider.Provider)
	}

	// 4. Get API key
	var apiKey struct {
		ID       int    `gorm:"column:id"`
		Provider string `gorm:"column:provider"`
		APIToken string `gorm:"column:api_token"`
		Account  string `gorm:"column:account"`
	}
	if err := db.DB.Table("api_keys").Where("id = ?", provider.APIKeyID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("api_key not found: %w", err)
	}

	// 5. Call Cloudflare API: List Records
	cfProvider := cloudflare.NewCloudflareProvider(apiKey.Account, apiKey.APIToken)
	records, err := cfProvider.ListRecords(ctx, provider.ProviderZoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cloudflare records: %w", err)
	}

	result := &PullSyncResult{
		Fetched: len(records),
	}

	// 6. Record sync start time
	syncStartedAt := time.Now()

	// 7. Build map of Cloudflare record IDs for quick lookup
	cfRecordIDs := make(map[string]bool)
	for _, record := range records {
		cfRecordIDs[record.ID] = true
	}

	// 8. Sync each Cloudflare record
	for _, record := range records {
		// Only sync A, AAAA, CNAME, TXT records
		if !isSupportedRecordType(record.Type) {
			continue
		}

		created, updated, err := syncSingleRecord(domainID, domain.Domain, record, syncStartedAt)
		if err != nil {
			log.Printf("[DNSPullSync] Failed to sync record %s: %v", record.ID, err)
			continue
		}

		if created {
			result.Created++
		} else if updated {
			result.Updated++
		}
	}

	// 9. Delete local records that are not in Cloudflare
	// Rule: If local record has provider_record_id but not in Cloudflare, delete it
	var localRecords []model.DomainDNSRecord
	if err := db.DB.Where("domain_id = ? AND provider_record_id IS NOT NULL AND provider_record_id != ''", domainID).Find(&localRecords).Error; err != nil {
		log.Printf("[DNSPullSync] Failed to query local records: %v", err)
	} else {
		for _, localRecord := range localRecords {
			if !cfRecordIDs[localRecord.ProviderRecordID] {
				// Record exists locally but not in Cloudflare, delete it
				if err := db.DB.Delete(&localRecord).Error; err != nil {
					log.Printf("[DNSPullSync] Failed to delete local record %d: %v", localRecord.ID, err)
				} else {
					log.Printf("[DNSPullSync] Deleted local record %d (provider_record_id=%s, not in Cloudflare)", 
						localRecord.ID, localRecord.ProviderRecordID)
					result.Deleted++
				}
			}
		}
	}

	return result, nil
}

// syncSingleRecord syncs a single Cloudflare record to local database
// Returns (created, updated, error)
// Core principle: provider_record_id is the unique identity
func syncSingleRecord(domainID int, zoneDomain string, record cloudflare.CloudflareRecord, syncStartedAt time.Time) (bool, bool, error) {
	// 1. Normalize name from Cloudflare (may be FQDN) to relative name
	normalizedName := NormalizeRelativeName(record.Name, zoneDomain)

	// 2. Try to find existing record by provider_record_id
	// Rule: record_id is the unique identity, not (name, type, value)
	var existingRecord model.DomainDNSRecord
	err := db.DB.Where("provider_record_id = ?", record.ID).First(&existingRecord).Error

	if err != nil {
		// Record does not exist locally (new record in Cloudflare)
		// Rule: Pull can INSERT new records from Cloudflare
		newRecord := model.DomainDNSRecord{
			DomainID:         domainID,
			Type:             model.DNSRecordType(record.Type),
			Name:             normalizedName,
			Value:            record.Content,
			TTL:              record.TTL,
			Proxied:          record.Proxied,
			Status:           model.DNSRecordStatusActive,
			DesiredState:     model.DNSRecordDesiredStatePresent,
			ProviderRecordID: record.ID,
			OwnerType:        model.DNSRecordOwnerExternal,
			OwnerID:          0,
			RetryCount:       0,
		}

		if err := db.DB.Create(&newRecord).Error; err != nil {
			return false, false, fmt.Errorf("failed to create record: %w", err)
		}

		log.Printf("[DNSPullSync] Created record: %s %s %s (provider_record_id=%s)", 
			record.Type, normalizedName, record.Content, record.ID)
		return true, false, nil
	}

	// 3. Record exists locally, UPDATE it
	// Rule: Same record_id = UPDATE (not delete + insert)
	// Update all fields from Cloudflare (Cloudflare is source of truth)
	updates := map[string]interface{}{
		"type":               model.DNSRecordType(record.Type),
		"name":               normalizedName,
		"value":              record.Content,
		"ttl":                record.TTL,
		"proxied":            record.Proxied,
		"status":             model.DNSRecordStatusActive,
		"desired_state":      model.DNSRecordDesiredStatePresent,
		"last_error":         nil,
		"provider_record_id": record.ID, // Ensure record_id is set
	}

	if err := db.DB.Model(&existingRecord).Updates(updates).Error; err != nil {
		return false, false, fmt.Errorf("failed to update record: %w", err)
	}

	log.Printf("[DNSPullSync] Updated record %d: %s %s %s (provider_record_id=%s)", 
		existingRecord.ID, record.Type, normalizedName, record.Content, record.ID)
	return false, true, nil
}

// isSupportedRecordType checks if the record type is supported
func isSupportedRecordType(recordType string) bool {
	switch recordType {
	case "A", "AAAA", "CNAME", "TXT":
		return true
	default:
		return false
	}
}
