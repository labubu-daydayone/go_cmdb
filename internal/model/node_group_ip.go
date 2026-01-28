package model

// NodeGroupIP represents the mapping between node group and IP
type NodeGroupIP struct {
	BaseModel
	NodeGroupID int `gorm:"uniqueIndex:uk_node_group_ips;not null" json:"node_group_id"`
	IPID        int `gorm:"uniqueIndex:uk_node_group_ips;index;not null" json:"ip_id"`
	
	// Relations
	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID" json:"node_group,omitempty"`
	IP        *NodeIP    `gorm:"foreignKey:IPID" json:"ip,omitempty"`
}

// TableName specifies the table name for NodeGroupIP model
func (NodeGroupIP) TableName() string {
	return "node_group_ips"
}
