package dto

import "time"

// DNSRecordDTO represents a DNS record in API responses
type DNSRecordDTO struct {
	ID               int       `json:"id"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	DomainId         int       `json:"domainId"`
	Type             string    `json:"type"`
	Name             string    `json:"name"`
	Value            string    `json:"value"`
	TTL              int       `json:"ttl"`
	Proxied          bool      `json:"proxied"`
	Status           string    `json:"status"`
	DesiredState     string    `json:"desiredState"`
	ProviderRecordId string    `json:"providerRecordId"`
	LastError        string    `json:"lastError"`
	RetryCount       int       `json:"retryCount"`
	NextRetryAt      *time.Time `json:"nextRetryAt"`
	OwnerType        string    `json:"ownerType"`
	OwnerId          int       `json:"ownerId"`
}
