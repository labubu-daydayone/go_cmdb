package domain

import (
	"context"
	"fmt"

	"go_cmdb/internal/db"
	"go_cmdb/internal/model"
)

// EnableCDN enables CDN for a domain by setting purpose to 'cdn'
// Validates:
// - domain exists
// - domain.status = active
// - domain has provider binding (domain_dns_providers exists)
// - domain has NS sync (domain_dns_zone_meta exists)
// - current purpose != cdn (idempotent)
func EnableCDN(ctx context.Context, domainID int) (*model.Domain, error) {
	// 1. Check if domain exists
	var domain model.Domain
	if err := db.DB.Where("id = ?", domainID).First(&domain).Error; err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	// 2. Check if status is active
	if domain.Status != model.DomainStatusActive {
		return nil, fmt.Errorf("domain status must be active, current: %s", domain.Status)
	}

	// 3. Check if domain has provider binding
	var providerCount int64
	if err := db.DB.Model(&model.DomainDNSProvider{}).
		Where("domain_id = ?", domainID).
		Count(&providerCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check provider binding: %w", err)
	}
	if providerCount == 0 {
		return nil, fmt.Errorf("domain has no DNS provider binding")
	}

	// 4. Check if domain has NS sync (domain_dns_zone_meta exists)
	var zoneMetaCount int64
	if err := db.DB.Table("domain_dns_zone_meta").
		Where("domain_id = ?", domainID).
		Count(&zoneMetaCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check zone meta: %w", err)
	}
	if zoneMetaCount == 0 {
		return nil, fmt.Errorf("domain has no NS sync data")
	}

	// 5. Check if already enabled (idempotent)
	if domain.Purpose == model.DomainPurposeCDN {
		return &domain, nil
	}

	// 6. Update purpose to cdn
	if err := db.DB.Model(&domain).Update("purpose", model.DomainPurposeCDN).Error; err != nil {
		return nil, fmt.Errorf("failed to update domain purpose: %w", err)
	}

	// Reload domain to get updated data
	if err := db.DB.Where("id = ?", domainID).First(&domain).Error; err != nil {
		return nil, fmt.Errorf("failed to reload domain: %w", err)
	}

	return &domain, nil
}

// DisableCDN disables CDN for a domain by setting purpose to 'general'
// Validates:
// - domain exists
// - current purpose = cdn
// - domain is not in use (not referenced by line_groups or websites)
func DisableCDN(ctx context.Context, domainID int) (*model.Domain, error) {
	// 1. Check if domain exists
	var domain model.Domain
	if err := db.DB.Where("id = ?", domainID).First(&domain).Error; err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	// 2. Check if current purpose is cdn
	if domain.Purpose != model.DomainPurposeCDN {
		return nil, fmt.Errorf("domain purpose must be cdn to disable, current: %s", domain.Purpose)
	}

	// 3. Check if domain is used by line_groups
	var lineGroupCount int64
	if err := db.DB.Table("line_groups").
		Where("domain_id = ?", domainID).
		Count(&lineGroupCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check line group usage: %w", err)
	}
	if lineGroupCount > 0 {
		return nil, fmt.Errorf("domain is in use by line group")
	}

	// 4. Check if domain is used by websites (through website_domains)
	var websiteDomainCount int64
	if err := db.DB.Table("website_domains").
		Where("domain_id = ?", domainID).
		Count(&websiteDomainCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check website usage: %w", err)
	}
	if websiteDomainCount > 0 {
		return nil, fmt.Errorf("domain is in use by website")
	}

	// 5. Update purpose to general
	if err := db.DB.Model(&domain).Update("purpose", model.DomainPurposeGeneral).Error; err != nil {
		return nil, fmt.Errorf("failed to update domain purpose: %w", err)
	}

	// Reload domain to get updated data
	if err := db.DB.Where("id = ?", domainID).First(&domain).Error; err != nil {
		return nil, fmt.Errorf("failed to reload domain: %w", err)
	}

	return &domain, nil
}
