package model

// CacheRuleItem 缓存规则项
type CacheRuleItem struct {
	BaseModel
	CacheRuleID int    `gorm:"index;not null;uniqueIndex:idx_cache_rule_item_unique" json:"cacheRuleId"`
	MatchType   string `gorm:"type:enum('path','suffix','exact');not null;uniqueIndex:idx_cache_rule_item_unique" json:"matchType"`
	MatchValue  string `gorm:"type:varchar(255);not null;uniqueIndex:idx_cache_rule_item_unique" json:"matchValue"`
	Mode        string `gorm:"type:enum('default','follow','force','bypass');not null;default:'default'" json:"mode"`
	TTLSeconds  int    `gorm:"not null" json:"ttlSeconds"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`

	// 关联
	CacheRule *CacheRule `gorm:"foreignKey:CacheRuleID" json:"cacheRule,omitempty"`
}

// TableName 指定表名
func (CacheRuleItem) TableName() string {
	return "cache_rule_items"
}

// MatchType constants
const (
	MatchTypePath   = "path"
	MatchTypeSuffix = "suffix"
	MatchTypeExact  = "exact"
)
