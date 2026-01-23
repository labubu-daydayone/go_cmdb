package model

// OriginGroupAddress 回源分组地址
type OriginGroupAddress struct {
	BaseModel
	OriginGroupID int    `gorm:"index;not null;uniqueIndex:idx_og_addr_unique" json:"origin_group_id"`
	Role          string `gorm:"type:enum('primary','backup');not null;uniqueIndex:idx_og_addr_unique" json:"role"`
	Protocol      string `gorm:"type:enum('http','https');not null;uniqueIndex:idx_og_addr_unique" json:"protocol"`
	Address       string `gorm:"type:varchar(255);not null;uniqueIndex:idx_og_addr_unique" json:"address"` // ip:port 或 domain:port
	Weight        int    `gorm:"default:10;not null" json:"weight"`
	Enabled       bool   `gorm:"default:true;not null" json:"enabled"`

	// 关联
	OriginGroup *OriginGroup `gorm:"foreignKey:OriginGroupID" json:"origin_group,omitempty"`
}

// TableName 指定表名
func (OriginGroupAddress) TableName() string {
	return "origin_group_addresses"
}

// Role constants
const (
	OriginRolePrimary = "primary"
	OriginRoleBackup  = "backup"
)

// Protocol constants
const (
	OriginProtocolHTTP  = "http"
	OriginProtocolHTTPS = "https"
)
