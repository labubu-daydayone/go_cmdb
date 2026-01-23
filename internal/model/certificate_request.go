package model

import "time"

// CertificateRequest represents a certificate request (ACME order)
type CertificateRequest struct {
	ID                  int        `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID           int        `gorm:"not null;index" json:"accountId"`
	Domains             string     `gorm:"type:text;not null" json:"domains"` // JSON array: ["example.com","*.example.com"]
	Status              string     `gorm:"type:varchar(20);not null;default:pending;index" json:"status"` // pending|running|success|failed
	Attempts            int        `gorm:"not null;default:0" json:"attempts"` // Retry attempts
	PollMaxAttempts     int        `gorm:"not null;default:10" json:"pollMaxAttempts"` // Max retry attempts
	LastError           string     `gorm:"type:text" json:"lastError"` // Last error message
	ResultCertificateID *int       `gorm:"index" json:"resultCertificateId"` // Reference to certificates.id
	RenewCertID         *int       `gorm:"index" json:"renewCertId"` // Certificate ID for renewal (null for new certificate)
	CreatedAt           time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt           time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
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
