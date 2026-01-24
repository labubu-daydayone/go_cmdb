package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/db"
	"go_cmdb/internal/dns/providers/cloudflare"
	"go_cmdb/internal/model"
)

// SyncResult represents the result of domain synchronization
type SyncResult struct {
	Total   int `json:"total"`
	Created int `json:"created"`
	Updated int `json:"updated"`
}

// SyncDomainsByAPIKey synchronizes domains from Cloudflare to local database
func SyncDomainsByAPIKey(ctx context.Context, apiKeyID int) (*SyncResult, error) {
	// 1. Validate apiKeyID exists
	var apiKey struct {
		ID       int    `gorm:"column:id"`
		Provider string `gorm:"column:provider"`
		APIToken string `gorm:"column:api_token"`
		Account  string `gorm:"column:account"`
	}
	if err := db.DB.Table("api_keys").Where("id = ?", apiKeyID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("api_key not found: %w", err)
	}

	// 2. Validate provider is cloudflare
	if apiKey.Provider != "cloudflare" {
		return nil, fmt.Errorf("api_key provider must be cloudflare, got: %s", apiKey.Provider)
	}

	// 3. Call Cloudflare API: List Zones
	cfProvider := cloudflare.NewCloudflareProvider(apiKey.Account, apiKey.APIToken)
	zones, err := cfProvider.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list cloudflare zones: %w", err)
	}

	result := &SyncResult{
		Total: len(zones),
	}

	// 4. Sync each zone
	for _, zone := range zones {
		created, err := syncSingleZone(apiKeyID, zone)
		if err != nil {
			log.Printf("[DomainSync] Failed to sync zone %s: %v", zone.Name, err)
			continue
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	return result, nil
}

// syncSingleZone syncs a single Cloudflare zone to local database
// Returns (created, error)
func syncSingleZone(apiKeyID int, zone cloudflare.Zone) (bool, error) {
	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Check if domain exists
	var existingDomain model.Domain
	err := tx.Where("domain = ?", zone.Name).First(&existingDomain).Error
	
	created := false
	var domainID int

	if err != nil {
		// Domain does not exist, create it with purpose=unset
		newDomain := model.Domain{
			Domain:  zone.Name,
			Purpose: model.DomainPurposeUnset,
			Status:  model.DomainStatusActive,
		}
		if err := tx.Create(&newDomain).Error; err != nil {
			tx.Rollback()
			return false, fmt.Errorf("failed to create domain: %w", err)
		}
		domainID = newDomain.ID
		created = true
	} else {
		// Domain exists, do not modify purpose
		domainID = existingDomain.ID
	}

	// 2. Check if domain is already bound to a non-cloudflare provider
	var existingProvider model.DomainDNSProvider
	err = tx.Where("domain_id = ?", domainID).First(&existingProvider).Error
	if err == nil {
		// Provider binding exists
		if existingProvider.Provider != model.DNSProviderCloudflare {
			// Skip if bound to non-cloudflare provider
			tx.Rollback()
			log.Printf("[DomainSync] Domain %s already bound to provider %s, skipping", zone.Name, existingProvider.Provider)
			return false, nil
		}
	}

	// 3. Upsert domain_dns_providers
	provider := model.DomainDNSProvider{
		DomainID:       domainID,
		Provider:       model.DNSProviderCloudflare,
		ProviderZoneID: zone.ID,
		APIKeyID:       apiKeyID,
		Status:         model.DNSProviderStatusActive,
	}

	// Try to update first
	result := tx.Model(&model.DomainDNSProvider{}).
		Where("domain_id = ?", domainID).
		Updates(map[string]interface{}{
			"provider":         provider.Provider,
			"provider_zone_id": provider.ProviderZoneID,
			"api_key_id":       provider.APIKeyID,
			"status":           provider.Status,
		})

	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to update domain_dns_provider: %w", result.Error)
	}

	// If no rows affected, create new record
	if result.RowsAffected == 0 {
		if err := tx.Create(&provider).Error; err != nil {
			tx.Rollback()
			return false, fmt.Errorf("failed to create domain_dns_provider: %w", err)
		}
	}

	// 4. Upsert domain_dns_zone_meta
	nameServersJSON, err := json.Marshal(zone.NameServers)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to marshal name_servers: %w", err)
	}

	// Try to update first
	result = tx.Table("domain_dns_zone_meta").
		Where("domain_id = ?", domainID).
		Updates(map[string]interface{}{
			"name_servers_json": nameServersJSON,
			"last_sync_at":      db.DB.NowFunc(),
			"last_error":        nil,
		})

	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to update domain_dns_zone_meta: %w", result.Error)
	}

	// If no rows affected, create new record
	if result.RowsAffected == 0 {
		zoneMeta := map[string]interface{}{
			"domain_id":         domainID,
			"name_servers_json": nameServersJSON,
			"last_sync_at":      db.DB.NowFunc(),
			"last_error":        nil,
		}
		if err := tx.Table("domain_dns_zone_meta").Create(zoneMeta).Error; err != nil {
			tx.Rollback()
			return false, fmt.Errorf("failed to create domain_dns_zone_meta: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return created, nil
}
