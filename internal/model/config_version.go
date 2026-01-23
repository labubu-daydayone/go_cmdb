package model

import "time"

// ConfigVersion represents a configuration version record
type ConfigVersion struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Version   int64     `gorm:"column:version;not null;uniqueIndex" json:"version"`
	NodeID    int       `gorm:"column:node_id;not null;index:idx_node_version" json:"nodeId"`
	Payload   string    `gorm:"column:payload;type:text;not null" json:"payload"`
	Status    string    `gorm:"column:status;type:varchar(20);not null;default:pending" json:"status"`
	AppliedAt *time.Time `gorm:"column:applied_at" json:"appliedAt,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP;index" json:"createdAt"`
}

// TableName specifies the table name for ConfigVersion
func (ConfigVersion) TableName() string {
	return "config_versions"
}

// ConfigVersionStatus constants
const (
	ConfigVersionStatusPending = "pending"
	ConfigVersionStatusApplied = "applied"
	ConfigVersionStatusFailed  = "failed"
)
