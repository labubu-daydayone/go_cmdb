package model

// DNSProvider represents DNS provider type
type DNSProvider string

const (
	DNSProviderCloudflare DNSProvider = "cloudflare"
	DNSProviderAliyun     DNSProvider = "aliyun"
	DNSProviderTencent    DNSProvider = "tencent"
	DNSProviderHuawei     DNSProvider = "huawei"
	DNSProviderManual     DNSProvider = "manual"
)

// DNSProviderStatus represents DNS provider status
type DNSProviderStatus string

const (
	DNSProviderStatusActive   DNSProviderStatus = "active"
	DNSProviderStatusInactive DNSProviderStatus = "inactive"
)

// DomainDNSProvider represents DNS provider configuration for a domain
type DomainDNSProvider struct {
	BaseModel
	DomainID       int               `gorm:"uniqueIndex;not null" json:"domain_id"`
	Provider       DNSProvider       `gorm:"type:varchar(32);not null" json:"provider"`
	ProviderZoneID string            `gorm:"type:varchar(128)" json:"provider_zone_id"`
	APIKeyID       int               `gorm:"index" json:"api_key_id"`
	Status         DNSProviderStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}

// TableName specifies the table name for DomainDNSProvider model
func (DomainDNSProvider) TableName() string {
	return "domain_dns_providers"
}
