package model

import "time"

// Certificate represents an SSL/TLS certificate
type Certificate struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Status    string    `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending|issued|expired|revoked
	CertPem   string    `gorm:"type:text;not null" json:"certPem"`
	KeyPem    string    `gorm:"type:text;not null" json:"keyPem"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name for Certificate
func (Certificate) TableName() string {
	return "certificates"
}

// Certificate status constants
const (
	CertificateStatusPending = "pending"
	CertificateStatusIssued  = "issued"
	CertificateStatusExpired = "expired"
	CertificateStatusRevoked = "revoked"
)
