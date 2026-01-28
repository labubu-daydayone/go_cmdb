package model

import "time"

// NodeStatus represents node status
type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

// Node represents an agent node
type Node struct {
	BaseModel
	Name             string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	MainIP           string     `gorm:"type:varchar(64);index;not null" json:"main_ip"`
	AgentPort        int        `gorm:"default:8080" json:"agent_port"`
	Enabled          bool       `gorm:"type:tinyint;default:1" json:"enabled"`
	Status           NodeStatus `gorm:"type:enum('online','offline','maintenance');default:'offline'" json:"status"`
	LastSeenAt       *time.Time `gorm:"type:datetime;null" json:"last_seen_at"`
	LastHealthError  *string    `gorm:"type:varchar(255);null" json:"last_health_error"`
	HealthFailCount  int        `gorm:"type:int;not null;default:0" json:"health_fail_count"`
	IPs              []NodeIP    `gorm:"foreignKey:NodeID;constraint:OnDelete:CASCADE" json:"ips,omitempty"`
}

// TableName specifies the table name for Node model
func (Node) TableName() string {
	return "nodes"
}
