package acme

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"go_cmdb/internal/dns"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// WorkerConfig defines ACME worker configuration
type WorkerConfig struct {
	Enabled     bool
	IntervalSec int
	BatchSize   int
}

// Worker handles ACME certificate requests
type Worker struct {
	db          *gorm.DB
	service     *Service
	dnsService  *dns.Service
	config      WorkerConfig
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewWorker creates a new ACME worker
func NewWorker(db *gorm.DB, dnsService *dns.Service, config WorkerConfig) *Worker {
	return &Worker{
		db:          db,
		service:     NewService(db),
		dnsService:  dnsService,
		config:      config,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Start starts the worker
func (w *Worker) Start() {
	if !w.config.Enabled {
		log.Println("[ACME Worker] Disabled, skipping")
		close(w.stoppedChan)
		return
	}

	log.Printf("[ACME Worker] Starting with interval=%ds, batch=%d\n", w.config.IntervalSec, w.config.BatchSize)

	go w.run()
}

// Stop stops the worker
func (w *Worker) Stop() {
	if !w.config.Enabled {
		return
	}

	log.Println("[ACME Worker] Stopping...")
	close(w.stopChan)
	<-w.stoppedChan
	log.Println("[ACME Worker] Stopped")
}

// run is the main worker loop
func (w *Worker) run() {
	defer close(w.stoppedChan)

	ticker := time.NewTicker(time.Duration(w.config.IntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	w.tick()

	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopChan:
			return
		}
	}
}

// tick processes a batch of certificate requests
func (w *Worker) tick() {
	requests, err := w.service.GetPendingRequests(w.config.BatchSize)
	if err != nil {
		log.Printf("[ACME Worker] Failed to get pending requests: %v\n", err)
		return
	}

	if len(requests) == 0 {
		return
	}

	log.Printf("[ACME Worker] Processing %d certificate requests\n", len(requests))

	for _, request := range requests {
		w.processRequest(&request)
	}
}

// processRequest processes a single certificate request
func (w *Worker) processRequest(request *model.CertificateRequest) {
	log.Printf("[ACME Worker] Processing request %d (attempts=%d)\n", request.ID, request.Attempts)

	// Step 1: Mark as running (optimistic lock)
	if err := w.service.MarkAsRunning(request.ID); err != nil {
		log.Printf("[ACME Worker] Request %d already processed: %v\n", request.ID, err)
		return
	}

	// Step 2: Get account
	var account model.AcmeAccount
	if err := w.db.First(&account, request.AccountID).Error; err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to get account: %v", err))
		return
	}

	// Step 3: Get provider
	var provider model.AcmeProvider
	if err := w.db.First(&provider, account.ProviderID).Error; err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	// Step 4: Parse domains
	var domains []string
	if err := json.Unmarshal([]byte(request.Domains), &domains); err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to parse domains: %v", err))
		return
	}

	// Step 5: Create lego client
	legoClient := NewLegoClient(w.db, w.dnsService, &provider, &account, request.ID)

	// Step 6: Ensure account is registered
	if err := legoClient.EnsureAccount(&account); err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to ensure account: %v", err))
		return
	}

	// Step 7: Request certificate
	result, err := legoClient.RequestCertificate(domains)
	if err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to request certificate: %v", err))
		return
	}

	// Step 8: Calculate certificate fingerprint
	fingerprint := calculateFingerprint(result.CertPem)

	// Step 9: Check if this is a renewal request (overwrite mode)
	if request.RenewCertID != nil && *request.RenewCertID > 0 {
		// Renewal mode: update existing certificate
		log.Printf("[ACME Worker] Renewal mode: updating certificate %d\n", *request.RenewCertID)
		
		// Update certificate record
		updates := map[string]interface{}{
			"fingerprint": fingerprint,
			"status":      model.CertificateStatusIssued,
			"cert_pem":    result.CertPem,
			"key_pem":     result.KeyPem,
			"chain_pem":   result.ChainPem,
			"issuer":      result.Issuer,
			"issue_at":    time.Now(),
			"expire_at":   extractExpiresAt(result.CertPem),
			"renewing":    false, // Clear renewing flag
			"last_error":  "",    // Clear error
		}
		
		if err := w.db.Model(&model.Certificate{}).Where("id = ?", *request.RenewCertID).Updates(updates).Error; err != nil {
			w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to update certificate: %v", err))
			// Clear renewing flag on failure
			w.db.Model(&model.Certificate{}).Where("id = ?", *request.RenewCertID).Update("renewing", false)
			return
		}
		
		// Delete old certificate_domains
		if err := w.db.Where("certificate_id = ?", *request.RenewCertID).Delete(&model.CertificateDomain{}).Error; err != nil {
			log.Printf("[ACME Worker] Failed to delete old certificate_domains: %v\n", err)
		}
		
		// Create new certificate_domains
		w.ensureCertificateDomains(*request.RenewCertID, domains)
		
		// Mark request as success
		if err := w.service.MarkAsSuccess(request.ID, *request.RenewCertID); err != nil {
			log.Printf("[ACME Worker] Failed to mark request as success: %v\n", err)
			return
		}
		
		log.Printf("[ACME Worker] Renewal request %d completed successfully, certificate_id=%d\n", request.ID, *request.RenewCertID)
		
		// Trigger website HTTPS apply and config apply with renew reason
		if err := w.service.OnCertificateIssued(request.ID, *request.RenewCertID); err != nil {
			log.Printf("[ACME Worker] Failed to trigger post-issuance actions: %v\n", err)
		}
		
		return
	}

	// Step 10: Check if certificate already exists (non-renewal mode)
	var existingCert model.Certificate
	err = w.db.Where("fingerprint = ?", fingerprint).First(&existingCert).Error
	if err == nil {
		// Certificate already exists, reuse it
		log.Printf("[ACME Worker] Certificate with fingerprint %s already exists (id=%d), reusing\n", fingerprint, existingCert.ID)
		
		// Update certificate_domains if needed
		w.ensureCertificateDomains(existingCert.ID, domains)
		
		// Mark request as success
		w.service.MarkAsSuccess(request.ID, existingCert.ID)
		
		// Trigger website HTTPS apply and config apply
		if err := w.service.OnCertificateIssued(request.ID, existingCert.ID); err != nil {
			log.Printf("[ACME Worker] Failed to trigger post-issuance actions: %v\n", err)
		}
		
		return
	}

	// Step 11: Create new certificate
	expireAt := extractExpiresAt(result.CertPem)
	issueAt := time.Now()
	acmeAccountID := request.AccountID
	certificate := &model.Certificate{
		Fingerprint:    fingerprint,
		Status:         "valid",
		CertificatePem: result.CertPem,
		PrivateKeyPem:  result.KeyPem,
		ExpireAt:       &expireAt,
		IssueAt:        &issueAt,
		Source:         "acme",
		Provider:       provider.Name,
		RenewMode:      "auto",
		AcmeAccountID:  &acmeAccountID,
	}

	if err := w.service.CreateCertificate(certificate); err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to create certificate: %v", err))
		return
	}

	// Step 12: Create certificate_domains
	w.ensureCertificateDomains(certificate.ID, domains)

	// Step 13: Mark request as success
	if err := w.service.MarkAsSuccess(request.ID, certificate.ID); err != nil {
		log.Printf("[ACME Worker] Failed to mark request as success: %v\n", err)
		return
	}

	log.Printf("[ACME Worker] Request %d completed successfully, certificate_id=%d\n", request.ID, certificate.ID)

	// Step 14: Trigger website HTTPS apply and config apply
	if err := w.service.OnCertificateIssued(request.ID, certificate.ID); err != nil {
		log.Printf("[ACME Worker] Failed to trigger post-issuance actions: %v\n", err)
	}
}

// ensureCertificateDomains ensures certificate_domains records exist
func (w *Worker) ensureCertificateDomains(certificateID int, domains []string) {
	for _, domain := range domains {
		isWildcard := strings.HasPrefix(domain, "*.")
		
		// Check if domain already exists
		var existing model.CertificateDomain
		err := w.db.Where("certificate_id = ? AND domain = ?", certificateID, domain).First(&existing).Error
		if err == nil {
			// Already exists
			continue
		}

		// Create new domain
		certDomain := model.CertificateDomain{
			CertificateID: certificateID,
			Domain:        domain,
			IsWildcard:    isWildcard,
		}

		if err := w.db.Create(&certDomain).Error; err != nil {
			log.Printf("[ACME Worker] Failed to create certificate_domain: %v\n", err)
		}
	}
}

// triggerWebsiteApply is deprecated, use service.OnCertificateIssued instead

// calculateFingerprint calculates SHA256 fingerprint of certificate
func calculateFingerprint(certPem string) string {
	hash := sha256.Sum256([]byte(certPem))
	return hex.EncodeToString(hash[:])
}

// extractExpiresAt extracts expiration time from certificate
func extractExpiresAt(certPem string) time.Time {
	// Simplified implementation: return 90 days from now
	// In production, should parse certificate and extract actual expiration time
	return time.Now().Add(90 * 24 * time.Hour)
}
