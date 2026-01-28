package model

// NodeIPType represents node IP type
type NodeIPType string

const (
	NodeIPTypeMain NodeIPType = "main"
	NodeIPTypeSub  NodeIPType = "sub"
)

// NodeIP represents an IP address for a node (main or sub)
type NodeIP struct {
	BaseModel
	NodeID  int        `gorm:"not null;index" json:"node_id"`
	IP      string     `gorm:"type:varchar(64);uniqueIndex:uk_node_ips_ip;not null" json:"ip"`
	IPType  NodeIPType `gorm:"type:enum('main','sub');not null;index" json:"ip_type"`
	Enabled bool       `gorm:"type:tinyint;default:1" json:"enabled"`
	
	// Relations
	Node *Node `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

// TableName specifies the table name for NodeIP model
func (NodeIP) TableName() string {
	return "node_ips"
}
