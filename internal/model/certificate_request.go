package model

import "time"

// CertificateRequest represents a certificate request (ACME order)
type CertificateRequest struct {
	ID                  int        `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID           int        `gorm:"column:acme_account_id;not null;index" json:"accountId"`
	Domains             string     `gorm:"column:domains_json;type:json;not null" json:"domains"` // JSON array: ["example.com","*.example.com"]
	Status              string     `gorm:"type:varchar(20);not null;default:pending;index" json:"status"` // pending|running|success|failed
	PollIntervalSec     int        `gorm:"column:poll_interval_sec;not null;default:40" json:"pollIntervalSec"` // Poll interval in seconds
	PollMaxAttempts     int        `gorm:"column:poll_max_attempts;not null;default:10" json:"pollMaxAttempts"` // Max retry attempts
	Attempts            int        `gorm:"not null;default:0" json:"attempts"` // Retry attempts
	LastError           *string    `gorm:"column:last_error;type:varchar(255)" json:"lastError"` // Last error message
	ResultCertificateID *int       `gorm:"column:result_certificate_id;index" json:"resultCertificateId"` // Reference to certificates.id
	RenewCertID         *int       `gorm:"-" json:"renewCertId"` // Certificate ID for renewal (not in DB yet, kept for code compatibility)
	CreatedAt           *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt           *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName specifies the table name for CertificateRequest
func (CertificateRequest) TableName() string {
	return "certificate_requests"
}

// CertificateRequest status constants
const (
	CertificateRequestStatusPending = "pending"
	CertificateRequestStatusRunning = "running"
	CertificateRequestStatusSuccess = "success"
	CertificateRequestStatusFailed  = "failed"
)
