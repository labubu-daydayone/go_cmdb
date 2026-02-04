package model

// CacheRule 缓存规则组
type CacheRule struct {
	BaseModel
	Name    string `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Enabled bool   `gorm:"default:true;not null" json:"enabled"`

	// 关联
	Items []CacheRuleItem `gorm:"foreignKey:CacheRuleID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// TableName 指定表名
func (CacheRule) TableName() string {
	return "cache_rules"
}
