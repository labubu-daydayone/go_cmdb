package model

import "time"

// AcmeProvider represents an ACME provider (Let's Encrypt, Google Public CA, etc.)
type AcmeProvider struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"` // letsencrypt, google
	DirectoryURL string   `gorm:"type:varchar(255);not null" json:"directoryUrl"`
	RequiresEAB bool      `gorm:"not null;default:false" json:"requiresEab"` // External Account Binding
	Status      string    `gorm:"type:varchar(20);not null;default:active" json:"status"` // active|inactive
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name for AcmeProvider
func (AcmeProvider) TableName() string {
	return "acme_providers"
}

// AcmeProvider status constants
const (
	AcmeProviderStatusActive   = "active"
	AcmeProviderStatusInactive = "inactive"
)
