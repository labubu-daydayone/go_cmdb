package cert

import (
	"go_cmdb/internal/model"
	"time"

	"gorm.io/gorm"
)

// DecisionResult represents the result of certificate decision
type DecisionResult struct {
	// CertFound indicates whether an existing certificate was found that covers all domains
	CertFound bool
	// CertificateID is the ID of the found certificate (0 if not found)
	CertificateID int
	// ACMETriggered indicates whether an ACME request was triggered
	ACMETriggered bool
	// Downgraded indicates whether HTTPS was downgraded (no cert, no ACME available)
	Downgraded bool
	// DowngradeReason explains why HTTPS was downgraded
	DowngradeReason string
}

// DecideCertificate implements the certificate decision logic:
// 1. Find existing valid certificate that fully covers all domains
// 2. If found -> use it (return cert ID)
// 3. If not found -> check if ACME is available and trigger request
// 4. If ACME not available -> downgrade (return downgraded=true)
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
	if err := triggerACMERequest(tx, websiteID, acmeAccountID, domains); err != nil {
		result.Downgraded = true
		result.DowngradeReason = "failed to trigger ACME request: " + err.Error()
		return result, nil
	}

	result.ACMETriggered = true

	// Also store ACME provider/account in website_https
	if err := updateWebsiteHTTPSACME(tx, websiteID, acmeProviderID, acmeAccountID); err != nil {
		// Non-fatal, log but continue
		_ = err
	}

	return result, nil
}

// findCoveringCertificate finds a valid certificate that fully covers all given domains
// Priority: prefer certificates with more domain coverage, then by latest expiry
func findCoveringCertificate(tx *gorm.DB, domains []string) (int, bool, error) {
	// Get all valid certificates (not expired)
	var certificates []model.Certificate
	if err := tx.Where("status = ? AND expire_at > ?", model.CertificateStatusValid, time.Now()).
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
		if err := tx.Where("certificate_id = ?", cert.ID).Find(&certDomains).Error; err != nil {
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

// FindDefaultACME finds the default active ACME provider and account
// Priority: google_publicca > letsencrypt > others
func FindDefaultACME(tx *gorm.DB) (int, int, error) {
	var providers []model.AcmeProvider
	if err := tx.Where("status = ?", "active").Find(&providers).Error; err != nil || len(providers) == 0 {
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
	if err := tx.Where("provider_id = ? AND status = ?", selectedProvider.ID, "active").First(&account).Error; err != nil {
		return 0, 0, err
	}

	return int(selectedProvider.ID), int(account.ID), nil
}

// triggerACMERequest creates a certificate request (idempotent)
func triggerACMERequest(tx *gorm.DB, websiteID int, acmeAccountID int, domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	// Build domains JSON
	domainsJSON := buildDomainsJSON(domains)

	// Check if there's already a pending/running request with same domains
	var existingRequest model.CertificateRequest
	err := tx.Where("acme_account_id = ? AND domains_json = ? AND status IN (?)",
		acmeAccountID, domainsJSON, []string{"pending", "running"}).
		First(&existingRequest).Error

	if err == nil {
		// Already exists, skip
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

	return tx.Create(&certRequest).Error
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
