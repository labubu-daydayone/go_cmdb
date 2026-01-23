package model

// LineGroupStatus represents line group status
type LineGroupStatus string

const (
	LineGroupStatusActive   LineGroupStatus = "active"
	LineGroupStatusInactive LineGroupStatus = "inactive"
)

// LineGroup represents a line group that references a node group
type LineGroup struct {
	BaseModel
	Name        string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	DomainID    int             `gorm:"not null;index" json:"domain_id"`
	NodeGroupID int             `gorm:"not null;index" json:"node_group_id"`
	CNAMEPrefix string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"cname_prefix"`
	CNAME       string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"cname"`
	Status      LineGroupStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	
	// Relations
	Domain    *Domain    `gorm:"foreignKey:DomainID" json:"domain,omitempty"`
	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID" json:"node_group,omitempty"`
}

// TableName specifies the table name for LineGroup model
func (LineGroup) TableName() string {
	return "line_groups"
}
