package model

import (
	"time"

	"gorm.io/datatypes"
)

// DomainDNSZoneMeta represents DNS zone metadata and NS cache
type DomainDNSZoneMeta struct {
	ID              int            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DomainID        int            `gorm:"column:domain_id;uniqueIndex;not null" json:"domain_id"`
	NameServersJSON datatypes.JSON `gorm:"column:name_servers_json;type:json;not null" json:"name_servers_json"`
	LastSyncAt      time.Time      `gorm:"column:last_sync_at;not null" json:"last_sync_at"`
	LastError       *string        `gorm:"column:last_error;type:varchar(255)" json:"last_error"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for DomainDNSZoneMeta model
func (DomainDNSZoneMeta) TableName() string {
	return "domain_dns_zone_meta"
}
