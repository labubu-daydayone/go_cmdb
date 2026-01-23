package model

import "time"

// CertificateBinding represents a binding between a certificate and a website
type CertificateBinding struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	CertificateID int       `gorm:"not null;index:idx_cert_website" json:"certificateId"`
	WebsiteID     int       `gorm:"not null;index:idx_cert_website;uniqueIndex:idx_website_unique" json:"websiteId"`
	Status        string    `gorm:"type:varchar(20);not null;default:inactive" json:"status"` // inactive|active
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name for CertificateBinding
func (CertificateBinding) TableName() string {
	return "certificate_bindings"
}

// CertificateBinding status constants
const (
	CertificateBindingStatusInactive = "inactive"
	CertificateBindingStatusActive   = "active"
)
