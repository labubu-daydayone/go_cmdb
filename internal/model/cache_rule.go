package model

// CacheRule 缓存规则组
type CacheRule struct {
	BaseModel
	Name        string `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`
	CachePolicy string `gorm:"type:enum('respect_origin','force_cache','default_cache');not null;default:'respect_origin'" json:"cachePolicy"`

	// 关联
	Items []CacheRuleItem `gorm:"foreignKey:CacheRuleID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// TableName 指定表名
func (CacheRule) TableName() string {
	return "cache_rules"
}

// CachePolicy constants
const (
	CachePolicyRespectOrigin = "respect_origin" // 追随后端
	CachePolicyForceCache    = "force_cache"    // 强制缓存
	CachePolicyDefaultCache  = "default_cache"  // 默认缓存（支持强刷绕过）
)
