package dns

import "go_cmdb/internal/dnstypes"

// Provider defines the interface for DNS providers
type Provider interface {
	// EnsureRecord ensures a DNS record exists with the correct values
	// If the record exists, it will be updated; otherwise, it will be created
	// Returns: providerRecordID, changed (true if created/updated), error
	EnsureRecord(zoneID string, record dnstypes.DNSRecord) (providerRecordID string, changed bool, err error)

	// DeleteRecord deletes a DNS record by its provider-specific ID
	DeleteRecord(zoneID string, providerRecordID string) error

	// FindRecord finds a DNS record by type, name, and value
	// Returns: providerRecordID, error (ErrNotFound if not found)
	FindRecord(zoneID string, recordType string, name string, value string) (providerRecordID string, err error)
}
