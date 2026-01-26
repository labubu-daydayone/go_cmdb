package cert

import (
	"log"
	"time"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// CleanerConfig defines certificate cleaner configuration
type CleanerConfig struct {
	Enabled        bool
	IntervalSec    int
	FailedKeepDays int
}

// Cleaner handles automatic cleanup of failed certificate requests
type Cleaner struct {
	db          *gorm.DB
	config      CleanerConfig
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewCleaner creates a new certificate cleaner
func NewCleaner(db *gorm.DB, config CleanerConfig) *Cleaner {
	return &Cleaner{
		db:          db,
		config:      config,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Start starts the cleaner
func (c *Cleaner) Start() {
	if !c.config.Enabled {
		log.Println("[Cert Cleaner] Disabled, skipping")
		close(c.stoppedChan)
		return
	}

	log.Printf("[Cert Cleaner] Starting with interval=%ds, keep_days=%d\n", c.config.IntervalSec, c.config.FailedKeepDays)

	go c.run()
}

// Stop stops the cleaner
func (c *Cleaner) Stop() {
	if !c.config.Enabled {
		return
	}

	log.Println("[Cert Cleaner] Stopping...")
	close(c.stopChan)
	<-c.stoppedChan
	log.Println("[Cert Cleaner] Stopped")
}

// run is the main cleaner loop
func (c *Cleaner) run() {
	defer close(c.stoppedChan)

	ticker := time.NewTicker(time.Duration(c.config.IntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	c.tick()

	for {
		select {
		case <-ticker.C:
			c.tick()
		case <-c.stopChan:
			return
		}
	}
}

// tick performs cleanup of old failed requests
func (c *Cleaner) tick() {
	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(c.config.FailedKeepDays) * 24 * time.Hour)

	// Delete failed requests older than cutoff time
	result := c.db.
		Where("status = ? AND updated_at < ?", model.CertificateRequestStatusFailed, cutoffTime).
		Delete(&model.CertificateRequest{})

	if result.Error != nil {
		log.Printf("[Cert Cleaner] Failed to clean old failed requests: %v\n", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[Cert Cleaner] Cleaned %d failed requests older than %v\n", result.RowsAffected, cutoffTime)
	}
}
