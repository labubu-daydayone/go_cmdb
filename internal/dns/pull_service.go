package dns

import (
	"context"
	"fmt"
	"log"

	"go_cmdb/internal/db"
	"go_cmdb/internal/dns/providers/cloudflare"
	"go_cmdb/internal/model"
)

// PullSyncResult represents the result of DNS records pull synchronization
type PullSyncResult struct {
	Fetched int `json:"fetched"` // Total records fetched from Cloudflare
	Created int `json:"created"` // New external records created
	Updated int `json:"updated"` // Existing external records updated
	Skipped int `json:"skipped"` // System-managed records skipped
}

// PullSyncRecords pulls DNS records from Cloudflare and syncs to local database
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

	// 6. Sync each record
	for _, record := range records {
		// Only sync A, AAAA, CNAME, TXT records
		if !isSupportedRecordType(record.Type) {
			continue
		}

		created, updated, skipped, err := syncSingleRecord(domainID, record)
		if err != nil {
			log.Printf("[DNSPullSync] Failed to sync record %s: %v", record.ID, err)
			continue
		}

		if created {
			result.Created++
		} else if updated {
			result.Updated++
		} else if skipped {
			result.Skipped++
		}
	}

	return result, nil
}

// syncSingleRecord syncs a single Cloudflare record to local database
// Returns (created, updated, skipped, error)
func syncSingleRecord(domainID int, record cloudflare.CloudflareRecord) (bool, bool, bool, error) {
	// 1. Check if record exists by provider_record_id
	var existingRecord model.DomainDNSRecord
	err := db.DB.Where("provider_record_id = ?", record.ID).First(&existingRecord).Error

	if err != nil {
		// Record does not exist, create new external record
		newRecord := model.DomainDNSRecord{
			DomainID:         domainID,
			Type:             model.DNSRecordType(record.Type),
			Name:             record.Name,
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
			return false, false, false, fmt.Errorf("failed to create external record: %w", err)
		}

		log.Printf("[DNSPullSync] Created external record: %s %s %s", record.Type, record.Name, record.Content)
		return true, false, false, nil
	}

	// Record exists, check owner_type
	if existingRecord.OwnerType != model.DNSRecordOwnerExternal {
		// System-managed record, skip
		log.Printf("[DNSPullSync] Skipped system-managed record: %s %s %s (owner_type=%s)", 
			record.Type, record.Name, record.Content, existingRecord.OwnerType)
		return false, false, true, nil
	}

	// External record exists, update if values changed
	needsUpdate := false
	updates := make(map[string]interface{})

	if existingRecord.Value != record.Content {
		updates["value"] = record.Content
		needsUpdate = true
	}
	if existingRecord.TTL != record.TTL {
		updates["ttl"] = record.TTL
		needsUpdate = true
	}
	if existingRecord.Proxied != record.Proxied {
		updates["proxied"] = record.Proxied
		needsUpdate = true
	}

	if needsUpdate {
		if err := db.DB.Model(&existingRecord).Updates(updates).Error; err != nil {
			return false, false, false, fmt.Errorf("failed to update external record: %w", err)
		}
		log.Printf("[DNSPullSync] Updated external record: %s %s %s", record.Type, record.Name, record.Content)
		return false, true, false, nil
	}

	// No update needed
	return false, false, false, nil
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
