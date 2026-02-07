package dns

import (
	"fmt"
	"go_cmdb/internal/domainutil"
	"go_cmdb/internal/model"
	"log"
	"strings"

	"gorm.io/gorm"
)

// EnsureWebsiteDomainCNAMEs creates CNAME records for all website domains
// pointing to the line group's CNAME.
//
// For each domain:
//   1. Calculate apex using PSL
//   2. Find domain_id from domains table (apex must be active)
//   3. Calculate CNAME name (subdomain part, or "@" for apex itself)
//   4. Calculate CNAME value (line_group cname_prefix + "." + apex domain)
//   5. Create domain_dns_record with owner_type=website_domain
//
// This function is idempotent: if a record already exists with the same
// (domain_id, type, name, owner_type, owner_id), it will be skipped.
func EnsureWebsiteDomainCNAMEs(tx *gorm.DB, websiteID int, domains []string, lineGroupID int) error {
	if len(domains) == 0 {
		return nil
	}

	// Get line group to find cname_prefix and domain_id
	var lineGroup struct {
		ID          int    `gorm:"column:id"`
		CnamePrefix string `gorm:"column:cname_prefix"`
		DomainID    int    `gorm:"column:domain_id"`
	}
	if err := tx.Table("line_groups").Where("id = ?", lineGroupID).First(&lineGroup).Error; err != nil {
		return fmt.Errorf("line group %d not found: %w", lineGroupID, err)
	}

	// Get the line group's apex domain name for constructing CNAME value
	var lineGroupDomain struct {
		Domain string `gorm:"column:domain"`
	}
	if err := tx.Table("domains").Where("id = ?", lineGroup.DomainID).First(&lineGroupDomain).Error; err != nil {
		return fmt.Errorf("domain %d not found for line group: %w", lineGroup.DomainID, err)
	}

	// CNAME value = cname_prefix + "." + line_group_domain
	cnameValue := lineGroup.CnamePrefix + "." + lineGroupDomain.Domain

	// Get website_domains to find owner_id for each domain
	var websiteDomains []model.WebsiteDomain
	if err := tx.Where("website_id = ?", websiteID).Find(&websiteDomains).Error; err != nil {
		return fmt.Errorf("failed to get website domains: %w", err)
	}

	// Build a map of domain -> website_domain.ID
	domainToWDID := make(map[string]int)
	for _, wd := range websiteDomains {
		domainToWDID[wd.Domain] = wd.ID
	}

	for _, domain := range domains {
		// Calculate apex
		apex, err := domainutil.EffectiveApex(domain)
		if err != nil {
			log.Printf("[DNS CNAME] Failed to calculate apex for %s: %v", domain, err)
			continue
		}

		// Find domain_id from domains table
		var domainRecord struct {
			ID int `gorm:"column:id"`
		}
		if err := tx.Table("domains").Where("domain = ? AND status = ?", apex, "active").First(&domainRecord).Error; err != nil {
			log.Printf("[DNS CNAME] Apex domain %s not found in domains table: %v", apex, err)
			continue
		}

		// Calculate CNAME name (subdomain part)
		cnameName := extractSubdomainName(domain, apex)

		// Find website_domain ID for owner_id
		wdID, ok := domainToWDID[domain]
		if !ok {
			log.Printf("[DNS CNAME] Website domain record not found for %s", domain)
			continue
		}

		// Also store CNAME in website_domains table
		if err := tx.Model(&model.WebsiteDomain{}).
			Where("id = ?", wdID).
			Update("cname", cnameValue).Error; err != nil {
			log.Printf("[DNS CNAME] Failed to update website_domain cname for %s: %v", domain, err)
		}

		// Check if record already exists (idempotent)
		var existingCount int64
		if err := tx.Table("domain_dns_records").
			Where("domain_id = ? AND type = ? AND name = ? AND owner_type = ? AND owner_id = ?",
				domainRecord.ID, model.DNSRecordTypeCNAME, cnameName,
				model.DNSRecordOwnerWebsiteDomain, wdID).
			Count(&existingCount).Error; err != nil {
			return fmt.Errorf("failed to check existing DNS record: %w", err)
		}

		if existingCount > 0 {
			// Update value if different
			if err := tx.Model(&model.DomainDNSRecord{}).
				Where("domain_id = ? AND type = ? AND name = ? AND owner_type = ? AND owner_id = ?",
					domainRecord.ID, model.DNSRecordTypeCNAME, cnameName,
					model.DNSRecordOwnerWebsiteDomain, wdID).
				Updates(map[string]interface{}{
					"value":         cnameValue,
					"desired_state": model.DNSRecordDesiredStatePresent,
					"status":        model.DNSRecordStatusPending,
				}).Error; err != nil {
				return fmt.Errorf("failed to update existing DNS record: %w", err)
			}
			log.Printf("[DNS CNAME] Updated existing CNAME record for %s -> %s", domain, cnameValue)
			continue
		}

		// Create new DNS record
		record := model.DomainDNSRecord{
			DomainID:     domainRecord.ID,
			Type:         model.DNSRecordTypeCNAME,
			Name:         cnameName,
			Value:        cnameValue,
			TTL:          120,
			Proxied:      false,
			Status:       model.DNSRecordStatusPending,
			DesiredState: model.DNSRecordDesiredStatePresent,
			OwnerType:    model.DNSRecordOwnerWebsiteDomain,
			OwnerID:      wdID,
		}

		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("failed to create DNS CNAME record for %s: %w", domain, err)
		}

		log.Printf("[DNS CNAME] Created CNAME record: %s -> %s (domain_id=%d, owner_id=%d)",
			cnameName, cnameValue, domainRecord.ID, wdID)
	}

	return nil
}

// extractSubdomainName extracts the subdomain part from a full domain
// relative to its apex domain.
//
// Examples:
//   - domain="www.wx39.xyz", apex="wx39.xyz" -> "www"
//   - domain="a.b.wx39.xyz", apex="wx39.xyz" -> "a.b"
//   - domain="wx39.xyz", apex="wx39.xyz" -> "@" (apex itself)
func extractSubdomainName(domain, apex string) string {
	if domain == apex {
		return "@"
	}

	// domain should end with ".apex"
	suffix := "." + apex
	if strings.HasSuffix(domain, suffix) {
		return strings.TrimSuffix(domain, suffix)
	}

	// Fallback: return the full domain
	return domain
}

// DeleteWebsiteDomainCNAMEs marks all CNAME records for a website's domains as absent
// This is used when a website is deleted or domains are changed
func DeleteWebsiteDomainCNAMEs(tx *gorm.DB, websiteID int) error {
	// Get all website_domain IDs
	var websiteDomains []model.WebsiteDomain
	if err := tx.Where("website_id = ?", websiteID).Find(&websiteDomains).Error; err != nil {
		return fmt.Errorf("failed to get website domains: %w", err)
	}

	if len(websiteDomains) == 0 {
		return nil
	}

	wdIDs := make([]int, len(websiteDomains))
	for i, wd := range websiteDomains {
		wdIDs[i] = wd.ID
	}

	// Mark all CNAME records as absent
	if err := tx.Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id IN ?", model.DNSRecordOwnerWebsiteDomain, wdIDs).
		Updates(map[string]interface{}{
			"desired_state": model.DNSRecordDesiredStateAbsent,
			"status":        model.DNSRecordStatusPending,
		}).Error; err != nil {
		return fmt.Errorf("failed to mark DNS records as absent: %w", err)
	}

	return nil
}
