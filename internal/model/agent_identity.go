package model

import "time"

// AgentIdentity represents an agent's identity bound to a certificate
type AgentIdentity struct {
	BaseModel
	NodeID          int       `gorm:"not null;uniqueIndex" json:"nodeId"`
	CertFingerprint string    `gorm:"type:varchar(128);not null;uniqueIndex" json:"certFingerprint"`
	Status          string    `gorm:"type:enum('active','revoked');default:'active';not null" json:"status"`
	IssuedAt        time.Time `gorm:"not null" json:"issuedAt"`
	RevokedAt       *time.Time `gorm:"null" json:"revokedAt,omitempty"`
}

// AgentIdentity status constants
const (
	AgentIdentityStatusActive  = "active"
	AgentIdentityStatusRevoked = "revoked"
)

func (AgentIdentity) TableName() string {
	return "agent_identities"
}
