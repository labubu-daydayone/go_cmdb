package model

import "time"

// AcmeAccount represents an ACME account for a specific provider
type AcmeAccount struct {
	ID              int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID      int       `gorm:"not null;index:idx_provider_email" json:"providerId"`
	Email           string    `gorm:"type:varchar(255);not null;index:idx_provider_email" json:"email"`
	AccountKeyPem   string    `gorm:"type:text;not null" json:"accountKeyPem"` // Private key for ACME account
	RegistrationURI string    `gorm:"type:varchar(500)" json:"registrationUri"` // ACME registration URI
	EabKid          string    `gorm:"type:varchar(255)" json:"eabKid"` // External Account Binding Key ID
	EabHmacKey      string    `gorm:"type:text" json:"eabHmacKey"` // External Account Binding HMAC Key (encrypted)
	EabExpiresAt    *time.Time `gorm:"" json:"eabExpiresAt"` // EAB expiration time (optional)
	Status          string    `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending|active|inactive
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name for AcmeAccount
func (AcmeAccount) TableName() string {
	return "acme_accounts"
}

// AcmeAccount status constants
const (
	AcmeAccountStatusPending  = "pending"
	AcmeAccountStatusActive   = "active"
	AcmeAccountStatusInactive = "inactive"
)
