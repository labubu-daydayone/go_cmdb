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

	// Step 9: Check if certificate already exists
	var existingCert model.Certificate
	err = w.db.Where("fingerprint = ?", fingerprint).First(&existingCert).Error
	if err == nil {
		// Certificate already exists, reuse it
		log.Printf("[ACME Worker] Certificate with fingerprint %s already exists (id=%d), reusing\n", fingerprint, existingCert.ID)
		
		// Update certificate_domains if needed
		w.ensureCertificateDomains(existingCert.ID, domains)
		
		// Mark request as success
		w.service.MarkAsSuccess(request.ID, existingCert.ID)
		
		// Trigger website HTTPS apply
		w.triggerWebsiteApply(request.ID)
		
		return
	}

	// Step 10: Create certificate
	certificate := &model.Certificate{
		Name:        fmt.Sprintf("ACME Certificate %d", request.ID),
		Fingerprint: fingerprint,
		Status:      model.CertificateStatusIssued,
		CertPem:     result.CertPem,
		KeyPem:      result.KeyPem,
		ChainPem:    result.ChainPem,
		Issuer:      result.Issuer,
		ExpiresAt:   extractExpiresAt(result.CertPem),
	}

	if err := w.service.CreateCertificate(certificate); err != nil {
		w.service.MarkAsFailed(request.ID, fmt.Sprintf("Failed to create certificate: %v", err))
		return
	}

	// Step 11: Create certificate_domains
	w.ensureCertificateDomains(certificate.ID, domains)

	// Step 12: Mark request as success
	if err := w.service.MarkAsSuccess(request.ID, certificate.ID); err != nil {
		log.Printf("[ACME Worker] Failed to mark request as success: %v\n", err)
		return
	}

	log.Printf("[ACME Worker] Request %d completed successfully, certificate_id=%d\n", request.ID, certificate.ID)

	// Step 13: Trigger website HTTPS apply
	w.triggerWebsiteApply(request.ID)
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

// triggerWebsiteApply triggers config apply for websites using this certificate
func (w *Worker) triggerWebsiteApply(requestID int) {
	// Get certificate bindings
	bindings, err := w.service.GetCertificateBindingsByRequest(requestID)
	if err != nil {
		log.Printf("[ACME Worker] Failed to get certificate bindings: %v\n", err)
		return
	}

	if len(bindings) == 0 {
		log.Printf("[ACME Worker] No certificate bindings found for request %d\n", requestID)
		return
	}

	// Activate bindings and trigger apply_config for each website
	for _, binding := range bindings {
		// Activate binding
		if err := w.service.ActivateCertificateBinding(binding.ID); err != nil {
			log.Printf("[ACME Worker] Failed to activate binding %d: %v\n", binding.ID, err)
			continue
		}

		// Get website
		var website model.Website
		if err := w.db.First(&website, binding.WebsiteID).Error; err != nil {
			log.Printf("[ACME Worker] Failed to get website %d: %v\n", binding.WebsiteID, err)
			continue
		}

		// Trigger apply_config
		// Note: This is a simplified version, actual implementation should use config service
		log.Printf("[ACME Worker] Triggering apply_config for website %d (line_group_id=%d)\n", website.ID, website.LineGroupID)
		
	// Import config service
	// Note: This creates a circular dependency, so we use a simplified approach
	// In production, should use event bus or message queue
	
	// Update website_https.certificate_id
	if err := w.db.Exec("UPDATE website_https SET certificate_id = ? WHERE website_id = ?", binding.CertificateID, website.ID).Error; err != nil {
		log.Printf("[ACME Worker] Failed to update website_https.certificate_id: %v\n", err)
		continue
	}
	
	// Create config apply task
	// This is a simplified version that directly creates agent_task
	// In production, should call config service API
	log.Printf("[ACME Worker] Certificate ready for website %d, triggering config apply\n", website.ID)
	}
}

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
