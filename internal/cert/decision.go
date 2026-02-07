package cert

import (
	"go_cmdb/internal/model"
	"log"
	"time"

	"gorm.io/gorm"
)

// DecisionResult represents the result of certificate decision
type DecisionResult struct {
	// CertFound indicates whether an existing certificate was found that covers all domains
	CertFound bool
	// CertificateID is the ID of the found certificate (0 if not found)
	CertificateID int
	// ACMENeeded indicates whether an ACME request is needed (replaces ACMETriggered)
	ACMENeeded bool
	// ACMEProviderID is the default ACME provider ID (set when ACMENeeded=true)
	ACMEProviderID int
	// ACMEAccountID is the default ACME account ID (set when ACMENeeded=true)
	ACMEAccountID int
	// Downgraded indicates whether HTTPS was downgraded (no cert, no ACME available)
	Downgraded bool
	// DowngradeReason explains why HTTPS was downgraded
	DowngradeReason string
	// ACMETriggered kept for backward compatibility
	ACMETriggered bool
}

// DecideCertificateReadOnly performs certificate decision without any DB writes.
// This is safe to call outside a transaction.
//
// Decision flow:
//  1. Find existing valid certificate that fully covers all domains
//  2. If found -> return cert ID (CertFound=true)
//  3. If not found -> check if ACME is available
//  4. If ACME available -> return ACMENeeded=true with provider/account IDs
//  5. If ACME not available -> return Downgraded=true
func DecideCertificateReadOnly(db *gorm.DB, domains []string) (*DecisionResult, error) {
	result := &DecisionResult{}

	if len(domains) == 0 {
		result.Downgraded = true
		result.DowngradeReason = "no domains provided"
		return result, nil
	}

	// Step 1: Find existing valid certificate that fully covers all domains
	log.Printf("[CertDecision] Searching for covering certificate for domains: %v", domains)
	certID, found, err := findCoveringCertificate(db, domains)
	if err != nil {
		return nil, err
	}

	if found {
		result.CertFound = true
		result.CertificateID = certID
		log.Printf("[CertDecision] Found covering certificate: id=%d", certID)
		return result, nil
	}
	log.Printf("[CertDecision] No covering certificate found")

	// Step 2: No existing cert found, check if ACME is available
	acmeProviderID, acmeAccountID, err := FindDefaultACME(db)
	if err != nil {
		// No ACME available -> downgrade
		result.Downgraded = true
		result.DowngradeReason = "no active ACME provider/account available"
		log.Printf("[CertDecision] No ACME available: %v", err)
		return result, nil
	}

	// ACME is available
	result.ACMENeeded = true
	result.ACMEProviderID = acmeProviderID
	result.ACMEAccountID = acmeAccountID
	log.Printf("[CertDecision] ACME needed: providerID=%d, accountID=%d", acmeProviderID, acmeAccountID)
	return result, nil
}

// DecideCertificate implements the certificate decision logic (legacy, writes to DB).
// Prefer DecideCertificateReadOnly + TriggerACMERequest for new code.
func DecideCertificate(tx *gorm.DB, websiteID int, domains []string) (*DecisionResult, error) {
	result := &DecisionResult{}

	if len(domains) == 0 {
		result.Downgraded = true
		result.DowngradeReason = "no domains provided"
		return result, nil
	}

	// Step 1: Find existing valid certificate that fully covers all domains
	certID, found, err := findCoveringCertificate(tx, domains)
	if err != nil {
		return nil, err
	}

	if found {
		result.CertFound = true
		result.CertificateID = certID
		return result, nil
	}

	// Step 2: No existing cert found, try ACME
	acmeProviderID, acmeAccountID, err := FindDefaultACME(tx)
	if err != nil {
		// No ACME available -> downgrade
		result.Downgraded = true
		result.DowngradeReason = "no active ACME provider/account available"
		return result, nil
	}

	// Step 3: Trigger ACME request
	if err := TriggerACMERequest(tx, websiteID, acmeAccountID, domains); err != nil {
		result.Downgraded = true
		result.DowngradeReason = "failed to trigger ACME request: " + err.Error()
		return result, nil
	}

	result.ACMETriggered = true
	result.ACMENeeded = true
	result.ACMEProviderID = acmeProviderID
	result.ACMEAccountID = acmeAccountID

	// Also store ACME provider/account in website_https
	if err := updateWebsiteHTTPSACME(tx, websiteID, acmeProviderID, acmeAccountID); err != nil {
		// Non-fatal, log but continue
		_ = err
	}

	return result, nil
}

// findCoveringCertificate finds a valid certificate that fully covers all given domains
// Priority: prefer certificates with more domain coverage, then by latest expiry
func findCoveringCertificate(db *gorm.DB, domains []string) (int, bool, error) {
	// Get all valid certificates (not expired, with >30 days remaining)
	minExpiry := time.Now().Add(30 * 24 * time.Hour)
	var certificates []model.Certificate
	if err := db.Where("status = ? AND expire_at > ?", model.CertificateStatusValid, minExpiry).
		Find(&certificates).Error; err != nil {
		return 0, false, err
	}

	if len(certificates) == 0 {
		return 0, false, nil
	}

	// For each certificate, check if it covers all domains
	var bestCertID int
	var bestExpireAt time.Time
	found := false

	for _, cert := range certificates {
		// Get certificate domains
		var certDomains []model.CertificateDomain
		if err := db.Where("certificate_id = ?", cert.ID).Find(&certDomains).Error; err != nil {
			continue
		}

		certDomainStrs := make([]string, len(certDomains))
		for i, cd := range certDomains {
			certDomainStrs[i] = cd.Domain
		}

		// Check coverage
		coverage := CalculateCoverage(certDomainStrs, domains)
		if coverage.Status == CoverageStatusCovered {
			// This cert covers all domains
			if !found || cert.ExpireAt.After(bestExpireAt) {
				bestCertID = cert.ID
				bestExpireAt = *cert.ExpireAt
				found = true
			}
		}
	}

	return bestCertID, found, nil
}

// FindDefaultACME finds the global default ACME provider and account
// Reads from acme_provider_defaults table (global unique default).
// Fallback: if no default set, prefer google_publicca > letsencrypt > others.
func FindDefaultACME(db *gorm.DB) (int, int, error) {
	// Step 1: Try to get the global default from acme_provider_defaults
	var defaultRecord model.ACMEProviderDefault
	if err := db.Order("id DESC").First(&defaultRecord).Error; err == nil {
		// Verify the account is still active
		var account model.AcmeAccount
		if err := db.Where("id = ? AND status = ?", defaultRecord.AccountID, "active").First(&account).Error; err == nil {
			return int(defaultRecord.ProviderID), int(defaultRecord.AccountID), nil
		}
	}

	// Step 2: Fallback - no valid default set, pick by provider priority
	var providers []model.AcmeProvider
	if err := db.Where("status = ?", "active").Find(&providers).Error; err != nil || len(providers) == 0 {
		if err != nil {
			return 0, 0, err
		}
		return 0, 0, gorm.ErrRecordNotFound
	}

	// Prefer google_publicca
	var selectedProvider *model.AcmeProvider
	for i := range providers {
		if providers[i].Name == "google_publicca" {
			selectedProvider = &providers[i]
			break
		}
	}
	if selectedProvider == nil {
		selectedProvider = &providers[0]
	}

	var account model.AcmeAccount
	if err := db.Where("provider_id = ? AND status = ?", selectedProvider.ID, "active").First(&account).Error; err != nil {
		return 0, 0, err
	}

	return int(selectedProvider.ID), int(account.ID), nil
}

// TriggerACMERequest creates a certificate request (idempotent).
// Exported so it can be called outside a transaction.
func TriggerACMERequest(db *gorm.DB, websiteID int, acmeAccountID int, domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	// Build domains JSON
	domainsJSON := buildDomainsJSON(domains)

	// Check if there's already a pending/running request with same domains
	var existingRequest model.CertificateRequest
	err := db.Where("acme_account_id = ? AND domains_json = ? AND status IN (?)",
		acmeAccountID, domainsJSON, []string{"pending", "running"}).
		First(&existingRequest).Error

	if err == nil {
		// Already exists, skip
		log.Printf("[ACME] Certificate request already exists (id=%d) for domains %s", existingRequest.ID, domainsJSON)
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// Create new certificate request
	certRequest := model.CertificateRequest{
		AccountID:       acmeAccountID,
		Domains:         domainsJSON,
		Status:          model.CertificateRequestStatusPending,
		PollIntervalSec: 40,
		PollMaxAttempts: 10,
		Attempts:        0,
	}

	if err := db.Create(&certRequest).Error; err != nil {
		return err
	}

	log.Printf("[ACME] Created certificate request (id=%d) for domains %s", certRequest.ID, domainsJSON)
	return nil
}

// updateWebsiteHTTPSACME updates the ACME provider/account in website_https
func updateWebsiteHTTPSACME(tx *gorm.DB, websiteID int, acmeProviderID int, acmeAccountID int) error {
	return tx.Model(&model.WebsiteHTTPS{}).
		Where("website_id = ?", websiteID).
		Updates(map[string]interface{}{
			"acme_provider_id": acmeProviderID,
			"acme_account_id":  acmeAccountID,
		}).Error
}

// buildDomainsJSON builds a JSON array string from domains
func buildDomainsJSON(domains []string) string {
	result := "["
	for i, domain := range domains {
		if i > 0 {
			result += ","
		}
		result += "\"" + domain + "\""
	}
	result += "]"
	return result
}
