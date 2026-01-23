package model

// NodeGroupStatus represents node group status
type NodeGroupStatus string

const (
	NodeGroupStatusActive   NodeGroupStatus = "active"
	NodeGroupStatusInactive NodeGroupStatus = "inactive"
)

// NodeGroup represents a group of node sub IPs
type NodeGroup struct {
	BaseModel
	Name        string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Description string          `gorm:"type:varchar(255)" json:"description"`
	DomainID    int             `gorm:"not null;index" json:"domain_id"`
	CNAMEPrefix string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"cname_prefix"`
	CNAME       string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"cname"`
	Status      NodeGroupStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	
	// Relations
	Domain *Domain              `gorm:"foreignKey:DomainID" json:"domain,omitempty"`
	SubIPs []NodeGroupSubIP     `gorm:"foreignKey:NodeGroupID;constraint:OnDelete:CASCADE" json:"sub_ips,omitempty"`
}

// TableName specifies the table name for NodeGroup model
func (NodeGroup) TableName() string {
	return "node_groups"
}
