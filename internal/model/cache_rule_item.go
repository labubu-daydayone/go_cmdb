package model

// CacheRuleItem 缓存规则项
type CacheRuleItem struct {
	BaseModel
	CacheRuleID int    `gorm:"index;not null;uniqueIndex:idx_cache_rule_item_unique" json:"cache_rule_id"`
	MatchType   string `gorm:"type:enum('path','suffix','exact');not null;uniqueIndex:idx_cache_rule_item_unique" json:"match_type"`
	MatchValue  string `gorm:"type:varchar(255);not null;uniqueIndex:idx_cache_rule_item_unique" json:"match_value"`
	TTLSeconds  int    `gorm:"not null" json:"ttl_seconds"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`

	// 关联
	CacheRule *CacheRule `gorm:"foreignKey:CacheRuleID" json:"cache_rule,omitempty"`
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
