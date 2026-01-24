package dns

import (
	"fmt"
	"log"
	"time"

	"go_cmdb/internal/dns/providers/cloudflare"
	"go_cmdb/internal/dnstypes"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// WorkerConfig holds configuration for DNS Worker
type WorkerConfig struct {
	Enabled      bool
	IntervalSec  int
	BatchSize    int
}

// Worker periodically syncs DNS records to Cloudflare
type Worker struct {
	db      *gorm.DB
	service *Service
	config  WorkerConfig
	stopCh  chan struct{}
}

// NewWorker creates a new DNS Worker
func NewWorker(db *gorm.DB, config WorkerConfig) *Worker {
	return &Worker{
		db:      db,
		service: NewService(db),
		config:  config,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the DNS Worker
func (w *Worker) Start() {
	if !w.config.Enabled {
		log.Println("[DNS Worker] Disabled, not starting")
		return
	}

	log.Printf("[DNS Worker] Starting with interval=%ds, batch_size=%d\n", 
		w.config.IntervalSec, w.config.BatchSize)

	go w.run()
}

// Stop stops the DNS Worker
func (w *Worker) Stop() {
	log.Println("[DNS Worker] Stopping...")
	close(w.stopCh)
}

// run is the main worker loop
func (w *Worker) run() {
	ticker := time.NewTicker(time.Duration(w.config.IntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	w.tick()

	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopCh:
			log.Println("[DNS Worker] Stopped")
			return
		}
	}
}

// tick processes one batch of DNS records
func (w *Worker) tick() {
	log.Println("[DNS Worker] Tick: processing DNS records...")

	// Step 1: Process pending/error records (desired_state=present)
	w.processPendingRecords()

	// Step 2: Process deletion records (desired_state=absent)
	w.processDeletionRecords()

	log.Println("[DNS Worker] Tick: done")
}

// processPendingRecords processes pending/error records
func (w *Worker) processPendingRecords() {
	records, err := w.service.GetPendingRecords(w.config.BatchSize)
	if err != nil {
		log.Printf("[DNS Worker] Failed to get pending records: %v\n", err)
		return
	}

	if len(records) == 0 {
		return
	}

	log.Printf("[DNS Worker] Processing %d pending records\n", len(records))

	for _, record := range records {
		w.processRecord(&record)
	}
}

// processDeletionRecords processes deletion records
func (w *Worker) processDeletionRecords() {
	records, err := w.service.GetDeletionRecords(w.config.BatchSize)
	if err != nil {
		log.Printf("[DNS Worker] Failed to get deletion records: %v\n", err)
		return
	}

	if len(records) == 0 {
		return
	}

	log.Printf("[DNS Worker] Processing %d deletion records\n", len(records))

	for _, record := range records {
		w.deleteRecord(&record)
	}
}

// processRecord processes a single DNS record (create/update)
func (w *Worker) processRecord(record *model.DomainDNSRecord) {
	log.Printf("[DNS Worker] Processing record %d (type=%s, name=%s, value=%s)\n", 
		record.ID, record.Type, record.Name, record.Value)

	// Step 1: Mark as running (optimistic locking)
	if err := w.service.MarkAsRunning(int(record.ID)); err != nil {
		log.Printf("[DNS Worker] Record %d: already being processed, skipping\n", record.ID)
		return
	}

	// Step 2: Get domain and provider info
	domain, err := w.service.GetDomain(record.DomainID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get domain: %v", err)
		log.Printf("[DNS Worker] Record %d: %s\n", record.ID, errMsg)
		w.service.MarkAsError(int(record.ID), errMsg)
		return
	}

	provider, err := w.service.GetDomainProvider(record.DomainID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get DNS provider: %v", err)
		log.Printf("[DNS Worker] Record %d: %s\n", record.ID, errMsg)
		w.service.MarkAsError(int(record.ID), errMsg)
		return
	}

	// Step 3: Get API token (decrypt if needed)
	// TODO: Implement decryption if api_keys.api_token is encrypted
	var apiKey model.APIKey
	if err := w.db.First(&apiKey, provider.APIKeyID).Error; err != nil {
		errMsg := fmt.Sprintf("failed to get API key: %v", err)
		log.Printf("[DNS Worker] Record %d: %s\n", record.ID, errMsg)
		w.service.MarkAsError(int(record.ID), errMsg)
		return
	}

	// Step 4: Create Cloudflare provider
	cfProvider := cloudflare.NewCloudflareProvider(apiKey.Account, apiKey.APIToken)

	// Step 5: Convert name to FQDN
	fqdn := ToFQDN(domain.Domain, record.Name)

	// Step 6: Ensure record in Cloudflare
	dnsRecord := dnstypes.DNSRecord{
		Type:    string(record.Type),
		Name:    fqdn,
		Value:   record.Value,
		TTL:     record.TTL,
		Proxied: record.Proxied,
	}

	providerRecordID, changed, err := cfProvider.EnsureRecord(provider.ProviderZoneID, dnsRecord)
	if err != nil {
		errMsg := fmt.Sprintf("cloudflare API error: %v", err)
		log.Printf("[DNS Worker] Record %d: %s\n", record.ID, errMsg)
		w.service.MarkAsError(int(record.ID), errMsg)
		return
	}

	// Step 7: Mark as active
	if err := w.service.MarkAsActive(int(record.ID), providerRecordID); err != nil {
		log.Printf("[DNS Worker] Record %d: failed to mark as active: %v\n", record.ID, err)
		return
	}

	if changed {
		log.Printf("[DNS Worker] Record %d: synced to Cloudflare (provider_record_id=%s, changed=true)\n", 
			record.ID, providerRecordID)
	} else {
		log.Printf("[DNS Worker] Record %d: already in sync (provider_record_id=%s, changed=false)\n", 
			record.ID, providerRecordID)
	}
}

// deleteRecord deletes a single DNS record from Cloudflare and local database
func (w *Worker) deleteRecord(record *model.DomainDNSRecord) {
	log.Printf("[DNS Worker] Deleting record %d (type=%s, name=%s, provider_record_id=%s)\n", 
		record.ID, record.Type, record.Name, record.ProviderRecordID)

	// Step 1: Get domain and provider info
	provider, err := w.service.GetDomainProvider(record.DomainID)
	if err != nil {
		log.Printf("[DNS Worker] Record %d: failed to get DNS provider: %v, deleting local record anyway\n", 
			record.ID, err)
		w.service.DeleteRecord(int(record.ID))
		return
	}

	// Step 2: Get API token
	var apiKey model.APIKey
	if err := w.db.First(&apiKey, provider.APIKeyID).Error; err != nil {
		log.Printf("[DNS Worker] Record %d: failed to get API key: %v, deleting local record anyway\n", 
			record.ID, err)
		w.service.DeleteRecord(int(record.ID))
		return
	}

	// Step 3: Create Cloudflare provider
	cfProvider := cloudflare.NewCloudflareProvider(apiKey.Account, apiKey.APIToken)

	// Step 4: Delete from Cloudflare
	if record.ProviderRecordID != "" {
		if err := cfProvider.DeleteRecord(provider.ProviderZoneID, record.ProviderRecordID); err != nil {
			log.Printf("[DNS Worker] Record %d: failed to delete from Cloudflare: %v, deleting local record anyway\n", 
				record.ID, err)
		} else {
			log.Printf("[DNS Worker] Record %d: deleted from Cloudflare\n", record.ID)
		}
	}

	// Step 5: Delete from local database (hard delete)
	if err := w.service.DeleteRecord(int(record.ID)); err != nil {
		log.Printf("[DNS Worker] Record %d: failed to delete from database: %v\n", record.ID, err)
		return
	}

	log.Printf("[DNS Worker] Record %d: deleted from local database\n", record.ID)
}
