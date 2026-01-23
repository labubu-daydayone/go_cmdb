package model

// OriginAddress 快照地址（属于origin_set）
// 允许相同IP在不同网站、不同set中重复出现
type OriginAddress struct {
	BaseModel
	OriginSetID int    `gorm:"index;not null;uniqueIndex:idx_oa_unique" json:"origin_set_id"`
	Role        string `gorm:"type:enum('primary','backup');not null;uniqueIndex:idx_oa_unique" json:"role"`
	Protocol    string `gorm:"type:enum('http','https');not null;uniqueIndex:idx_oa_unique" json:"protocol"`
	Address     string `gorm:"type:varchar(255);not null;uniqueIndex:idx_oa_unique" json:"address"` // ip:port 或 domain:port
	Weight      int    `gorm:"default:10;not null;uniqueIndex:idx_oa_unique" json:"weight"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`

	// 关联
	OriginSet *OriginSet `gorm:"foreignKey:OriginSetID" json:"origin_set,omitempty"`
}

// TableName 指定表名
func (OriginAddress) TableName() string {
	return "origin_addresses"
}
