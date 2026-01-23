package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// RiskType 风险类型
type RiskType string

const (
	RiskTypeDomainMismatch   RiskType = "domain_mismatch"   // 域名不匹配
	RiskTypeCertExpiring     RiskType = "cert_expiring"     // 证书即将过期
	RiskTypeACMERenewFailed  RiskType = "acme_renew_failed" // ACME续期失败
	RiskTypeWeakCoverage     RiskType = "weak_coverage"     // 弱覆盖
)

// RiskLevel 风险级别
type RiskLevel string

const (
	RiskLevelInfo     RiskLevel = "info"
	RiskLevelWarning  RiskLevel = "warning"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskStatus 风险状态
type RiskStatus string

const (
	RiskStatusActive   RiskStatus = "active"
	RiskStatusResolved RiskStatus = "resolved"
)

// RiskDetail 风险详情（JSON字段）
type RiskDetail map[string]interface{}

// Value implements driver.Valuer interface
func (d RiskDetail) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan implements sql.Scanner interface
func (d *RiskDetail) Scan(value interface{}) error {
	if value == nil {
		*d = make(RiskDetail)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, d)
}

// CertificateRisk 证书与网站风险记录
type CertificateRisk struct {
	ID            int         `gorm:"primaryKey;autoIncrement" json:"id"`
	RiskType      RiskType    `gorm:"type:enum('domain_mismatch','cert_expiring','acme_renew_failed','weak_coverage');not null" json:"risk_type"`
	Level         RiskLevel   `gorm:"type:enum('info','warning','critical');not null" json:"level"`
	CertificateID *int        `gorm:"index" json:"certificate_id,omitempty"`
	WebsiteID     *int        `gorm:"index" json:"website_id,omitempty"`
	Detail        RiskDetail  `gorm:"type:json;not null" json:"detail"`
	Status        RiskStatus  `gorm:"type:enum('active','resolved');not null;default:'active';index" json:"status"`
	DetectedAt    time.Time   `gorm:"not null;index" json:"detected_at"`
	ResolvedAt    *time.Time  `gorm:"index" json:"resolved_at,omitempty"`
	CreatedAt     time.Time   `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time   `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (CertificateRisk) TableName() string {
	return "certificate_risks"
}
