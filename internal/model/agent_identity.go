package model

import "time"

// AgentIdentityStatus represents the status of an agent identity
type AgentIdentityStatus string

const (
	AgentIdentityStatusActive  AgentIdentityStatus = "active"
	AgentIdentityStatusRevoked AgentIdentityStatus = "revoked"
)

// AgentIdentity represents an mTLS client certificate for a Node
type AgentIdentity struct {
	ID          int                 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	NodeID      int                 `gorm:"column:node_id;not null;uniqueIndex:uk_node_id" json:"nodeId"`
	Fingerprint string              `gorm:"column:fingerprint;type:varchar(128);not null;uniqueIndex:uk_fingerprint" json:"fingerprint"`
	CertPEM     string              `gorm:"column:cert_pem;type:longtext;not null" json:"certPem"`
	KeyPEM      string              `gorm:"column:key_pem;type:longtext;not null" json:"keyPem"`
	Status      AgentIdentityStatus `gorm:"column:status;type:enum('active','revoked');not null;default:'active'" json:"status"`
	IssuedAt    *time.Time          `gorm:"column:issued_at" json:"issuedAt"`
	RevokedAt   *time.Time          `gorm:"column:revoked_at" json:"revokedAt"`
	CreatedAt   *time.Time          `gorm:"column:created_at;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt   *time.Time          `gorm:"column:updated_at;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName specifies the table name for AgentIdentity
func (AgentIdentity) TableName() string {
	return "agent_identities"
}
