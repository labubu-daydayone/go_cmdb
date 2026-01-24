package model

import (
	"time"
)

// DNSRecordType represents DNS record type
type DNSRecordType string

const (
	DNSRecordTypeA     DNSRecordType = "A"
	DNSRecordTypeAAAA  DNSRecordType = "AAAA"
	DNSRecordTypeCNAME DNSRecordType = "CNAME"
	DNSRecordTypeTXT   DNSRecordType = "TXT"
)

// DNSRecordStatus represents DNS record status
type DNSRecordStatus string

const (
	DNSRecordStatusPending DNSRecordStatus = "pending"
	DNSRecordStatusActive  DNSRecordStatus = "active"
	DNSRecordStatusError   DNSRecordStatus = "error"
	DNSRecordStatusRunning DNSRecordStatus = "running" // Worker processing
)

// DNSRecordOwnerType represents DNS record owner type
type DNSRecordOwnerType string

const (
	DNSRecordOwnerNodeGroup      DNSRecordOwnerType = "node_group"
	DNSRecordOwnerLineGroup      DNSRecordOwnerType = "line_group"
	DNSRecordOwnerWebsiteDomain  DNSRecordOwnerType = "website_domain"
	DNSRecordOwnerACMEChallenge  DNSRecordOwnerType = "acme_challenge"
	DNSRecordOwnerExternal       DNSRecordOwnerType = "external" // External records from Cloudflare pull sync
)

// DNSRecordDesiredState represents desired state of DNS record
type DNSRecordDesiredState string

const (
	DNSRecordDesiredStatePresent DNSRecordDesiredState = "present" // Should exist in Cloudflare
	DNSRecordDesiredStateAbsent  DNSRecordDesiredState = "absent"  // Should be deleted from Cloudflare
)

// DomainDNSRecord represents a DNS record
type DomainDNSRecord struct {
	BaseModel
	DomainID         int                `gorm:"index:idx_domain_type_name;not null" json:"domain_id"`
	Type             DNSRecordType      `gorm:"type:enum('A','AAAA','CNAME','TXT');index:idx_domain_type_name;not null" json:"type"`
	Name             string             `gorm:"type:varchar(255);index:idx_domain_type_name;not null" json:"name"`
	Value            string             `gorm:"type:varchar(2048);not null" json:"value"`
	TTL              int                `gorm:"default:120" json:"ttl"`
	Proxied          bool               `gorm:"type:tinyint;default:0" json:"proxied"`
	Status           DNSRecordStatus    `gorm:"type:enum('pending','active','error','running');default:'pending'" json:"status"`
	DesiredState     DNSRecordDesiredState `gorm:"type:enum('present','absent');not null;default:'present'" json:"desired_state"`
	ProviderRecordID string             `gorm:"type:varchar(128)" json:"provider_record_id"`
	LastError        string             `gorm:"type:varchar(255)" json:"last_error"`
	RetryCount       int                `gorm:"default:0" json:"retry_count"`
	NextRetryAt      *time.Time         `json:"next_retry_at"`
	OwnerType        DNSRecordOwnerType `gorm:"type:enum('node_group','line_group','website_domain','acme_challenge','external');index:idx_owner;not null" json:"owner_type"`
	OwnerID          int                `gorm:"index:idx_owner;not null" json:"owner_id"`
}

// TableName specifies the table name for DomainDNSRecord model
func (DomainDNSRecord) TableName() string {
	return "domain_dns_records"
}
