package model

import "time"

// LineGroup represents a line group for CDN line management
type LineGroup struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	Name         string    `gorm:"column:name;type:varchar(128);not null" json:"-"`
	Description  string    `gorm:"column:description;type:varchar(255);not null;default:''" json:"-"`
	DomainID     int64     `gorm:"column:domain_id;not null;index:idx_domain_id" json:"-"`
	NodeGroupID  int64     `gorm:"column:node_group_id;not null;index:idx_node_group_id" json:"-"`
	CNAMEPrefix  string    `gorm:"column:cname_prefix;type:varchar(64);not null;uniqueIndex:uk_cname_prefix" json:"-"`
	Status       string    `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"-"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"-"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null" json:"-"`
	
	// Associations
	Domain    *Domain    `gorm:"foreignKey:DomainID" json:"-"`
	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID" json:"-"`
}

// TableName specifies the table name for LineGroup model
func (LineGroup) TableName() string {
	return "line_groups"
}

// LineGroupStatus constants
const (
	LineGroupStatusActive   = "active"
	LineGroupStatusDisabled = "disabled"
)
