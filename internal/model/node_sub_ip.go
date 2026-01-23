package model

// NodeSubIP represents a sub IP address for a node
type NodeSubIP struct {
	BaseModel
	NodeID  int    `gorm:"index:idx_node_ip;not null" json:"node_id"`
	IP      string `gorm:"type:varchar(64);index:idx_node_ip,idx_ip;not null" json:"ip"`
	Enabled bool   `gorm:"type:tinyint;default:1" json:"enabled"`
}

// TableName specifies the table name for NodeSubIP model
func (NodeSubIP) TableName() string {
	return "node_sub_ips"
}
