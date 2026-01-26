package model

import "time"

// Certificate represents an SSL/TLS certificate
type Certificate struct {
	ID              int        `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Provider        string     `gorm:"column:provider;type:enum('letsencrypt','google_publicca','manual');not null" json:"provider"`
	Source          string     `gorm:"column:source;type:enum('acme','manual');not null" json:"source"`
	AcmeAccountID   *int       `gorm:"column:acme_account_id;index" json:"acmeAccountId"`
	Status          string     `gorm:"column:status;type:enum('valid','expiring','expired','revoked');not null;default:valid" json:"status"`
	IssueAt         *time.Time `gorm:"column:issue_at" json:"issueAt"`
	ExpireAt        *time.Time `gorm:"column:expire_at" json:"expireAt"`
	Fingerprint     string     `gorm:"column:fingerprint;type:varchar(128);uniqueIndex;not null" json:"fingerprint"`
	CertificatePem  string     `gorm:"column:certificate_pem;type:longtext;not null" json:"certificatePem"`
	PrivateKeyPem   string     `gorm:"column:private_key_pem;type:longtext;not null" json:"privateKeyPem"`
	RenewMode       string     `gorm:"column:renew_mode;type:enum('auto','manual');not null;default:manual" json:"renewMode"`
	RenewAt         *time.Time `gorm:"column:renew_at" json:"renewAt"`
	LastError       *string    `gorm:"column:last_error;type:varchar(255)" json:"lastError"`
	CreatedAt       *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       *time.Time `gorm:"column:updated_at" json:"updatedAt"`
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
