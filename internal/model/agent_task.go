package model

import "time"

// AgentTask represents a task to be executed by an agent
type AgentTask struct {
	BaseModel
	NodeID       uint       `gorm:"not null;index" json:"nodeId"`
	Type         string    `gorm:"type:enum('purge_cache','apply_config','reload');not null" json:"type"`
	Payload      string    `gorm:"type:json" json:"payload"`
	Status       string    `gorm:"type:enum('pending','running','success','failed');default:'pending';index" json:"status"`
	LastError    string    `gorm:"type:varchar(255)" json:"lastError,omitempty"`
	Attempts     int       `gorm:"-" json:"attempts,omitempty"`
	NextRetryAt  *time.Time `gorm:"-" json:"nextRetryAt,omitempty"`
	RequestID    string    `gorm:"-" json:"requestId,omitempty"`
}

// TableName specifies the table name for AgentTask
func (AgentTask) TableName() string {
	return "agent_tasks"
}

// Task type constants
const (
	TaskTypePurgeCache  = "purge_cache"
	TaskTypeApplyConfig = "apply_config"
	TaskTypeReload      = "reload"
)

// Task status constants
const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)
