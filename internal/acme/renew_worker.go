package acme

import (
	"log"
	"time"
)

// RenewWorkerConfig holds configuration for the renew worker
type RenewWorkerConfig struct {
	Enabled          bool // Whether the worker is enabled
	IntervalSec      int  // Polling interval in seconds
	BatchSize        int  // Batch size for processing
	RenewBeforeDays  int  // Renew certificates expiring within this many days
}

// RenewWorker polls for certificates that need renewal
type RenewWorker struct {
	renewService *RenewService
	config       RenewWorkerConfig
	stopChan     chan struct{}
}

// NewRenewWorker creates a new RenewWorker
func NewRenewWorker(renewService *RenewService, config RenewWorkerConfig) *RenewWorker {
	return &RenewWorker{
		renewService: renewService,
		config:       config,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the worker
func (w *RenewWorker) Start() {
	if !w.config.Enabled {
		log.Println("[RenewWorker] Disabled, not starting")
		return
	}

	log.Printf("[RenewWorker] Starting with interval=%ds, batch=%d, renewBefore=%d days\n",
		w.config.IntervalSec, w.config.BatchSize, w.config.RenewBeforeDays)

	go w.run()
}

// Stop stops the worker
func (w *RenewWorker) Stop() {
	log.Println("[RenewWorker] Stopping...")
	close(w.stopChan)
}

// run is the main worker loop
func (w *RenewWorker) run() {
	ticker := time.NewTicker(time.Duration(w.config.IntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	w.tick()

	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopChan:
			log.Println("[RenewWorker] Stopped")
			return
		}
	}
}

// tick processes one batch of renewal candidates
func (w *RenewWorker) tick() {
	log.Println("[RenewWorker] Tick: checking for renewal candidates...")

	// Get renewal candidates
	candidates, err := w.renewService.GetRenewCandidates(w.config.RenewBeforeDays, w.config.BatchSize)
	if err != nil {
		log.Printf("[RenewWorker] Failed to get renewal candidates: %v\n", err)
		return
	}

	if len(candidates) == 0 {
		log.Println("[RenewWorker] No renewal candidates found")
		return
	}

	log.Printf("[RenewWorker] Found %d renewal candidates\n", len(candidates))

	// Process each candidate
	for _, cert := range candidates {
		w.processRenewal(cert.ID)
	}
}

// processRenewal processes a single certificate renewal
func (w *RenewWorker) processRenewal(certID int) {
	log.Printf("[RenewWorker] Processing renewal for certificate %d\n", certID)

	// Step 1: Mark as renewing (optimistic lock)
	if err := w.renewService.MarkAsRenewing(certID); err != nil {
		log.Printf("[RenewWorker] Failed to mark certificate %d as renewing: %v\n", certID, err)
		return
	}

	// Step 2: Get certificate details
	cert, err := w.renewService.GetCertificate(certID)
	if err != nil {
		log.Printf("[RenewWorker] Failed to get certificate %d: %v\n", certID, err)
		w.renewService.ClearRenewing(certID)
		return
	}

	// Step 3: Get certificate domains
	domains, err := w.renewService.GetCertificateDomains(certID)
	if err != nil {
		log.Printf("[RenewWorker] Failed to get domains for certificate %d: %v\n", certID, err)
		w.renewService.ClearRenewing(certID)
		return
	}

	if len(domains) == 0 {
		log.Printf("[RenewWorker] Certificate %d has no domains, skipping\n", certID)
		w.renewService.ClearRenewing(certID)
		return
	}

	// Step 4: Validate acme_account_id
	if cert.AcmeAccountID == 0 {
		log.Printf("[RenewWorker] Certificate %d has no acme_account_id, skipping\n", certID)
		w.renewService.ClearRenewing(certID)
		return
	}

	// Step 5: Create renewal request
	request, err := w.renewService.CreateRenewRequest(certID, cert.AcmeAccountID, domains)
	if err != nil {
		log.Printf("[RenewWorker] Failed to create renewal request for certificate %d: %v\n", certID, err)
		w.renewService.ClearRenewing(certID)
		return
	}

	log.Printf("[RenewWorker] Created renewal request %d for certificate %d (domains: %v)\n",
		request.ID, certID, domains)

	// Note: The renewing flag will be cleared by ACME Worker when renewal succeeds or fails
}
