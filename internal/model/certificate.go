package model

import "time"

// Certificate represents an SSL/TLS certificate
type Certificate struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	Fingerprint   string    `gorm:"type:varchar(64);uniqueIndex" json:"fingerprint"` // SHA256 fingerprint
	Status        string    `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending|issued|expired|revoked|valid|expiring
	CertPem       string    `gorm:"type:text;not null" json:"certPem"`
	KeyPem        string    `gorm:"type:text;not null" json:"keyPem"`
	ChainPem      string    `gorm:"type:text" json:"chainPem"` // Certificate chain (intermediate + root)
	Issuer        string    `gorm:"type:varchar(255)" json:"issuer"` // Certificate issuer
	IssueAt       time.Time `gorm:"" json:"issueAt"` // Certificate issue time
	ExpireAt      time.Time `gorm:"not null" json:"expireAt"` // Certificate expiration time
	Source        string    `gorm:"type:varchar(20);not null;default:manual" json:"source"` // manual|acme
	RenewMode     string    `gorm:"type:varchar(20);not null;default:manual" json:"renewMode"` // manual|auto
	AcmeAccountID int       `gorm:"index" json:"acmeAccountId"` // ACME account ID for renewal
	Renewing      bool      `gorm:"type:tinyint(1);not null;default:0" json:"renewing"` // Renewal in progress flag
	LastError     string    `gorm:"type:varchar(500)" json:"lastError"` // Last error message
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name for Certificate
func (Certificate) TableName() string {
	return "certificates"
}

// Certificate status constants
const (
	CertificateStatusPending  = "pending"
	CertificateStatusIssued   = "issued"
	CertificateStatusExpired  = "expired"
	CertificateStatusRevoked  = "revoked"
	CertificateStatusValid    = "valid"
	CertificateStatusExpiring = "expiring"
)

// Certificate source constants
const (
	CertificateSourceManual = "manual"
	CertificateSourceAcme   = "acme"
)

// Certificate renew mode constants
const (
	CertificateRenewModeManual = "manual"
	CertificateRenewModeAuto   = "auto"
)
