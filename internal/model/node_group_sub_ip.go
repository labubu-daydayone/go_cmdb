package model

// NodeGroupSubIP represents the mapping between node group and sub IP
type NodeGroupSubIP struct {
	BaseModel
	NodeGroupID int `gorm:"index:idx_ng_subip;not null" json:"node_group_id"`
	SubIPID     int `gorm:"index:idx_ng_subip;not null" json:"sub_ip_id"`
	
	// Relations
	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID" json:"node_group,omitempty"`
	SubIP     *NodeSubIP `gorm:"foreignKey:SubIPID" json:"sub_ip,omitempty"`
}

// TableName specifies the table name for NodeGroupSubIP model
func (NodeGroupSubIP) TableName() string {
	return "node_group_sub_ips"
}
