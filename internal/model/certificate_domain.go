package model

import "time"

// CertificateDomain represents a domain covered by a certificate (SAN)
type CertificateDomain struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	CertificateID int       `gorm:"not null;index:idx_cert_domain" json:"certificateId"`
	Domain        string    `gorm:"type:varchar(255);not null;index:idx_cert_domain" json:"domain"` // example.com or *.example.com
	IsWildcard    bool      `gorm:"not null;default:false" json:"isWildcard"` // true if domain starts with *.
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName specifies the table name for CertificateDomain
func (CertificateDomain) TableName() string {
	return "certificate_domains"
}
