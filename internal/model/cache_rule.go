package model

// CacheRule 缓存规则组
type CacheRule struct {
	BaseModel
	Name        string `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`
	DefaultMode string `gorm:"type:enum('default','follow','force','bypass');not null;default:'default'" json:"defaultMode"`

	// 关联
	Items []CacheRuleItem `gorm:"foreignKey:CacheRuleID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// TableName 指定表名
func (CacheRule) TableName() string {
	return "cache_rules"
}

// CacheMode constants
const (
	CacheModeDefault = "default" // 最简单缓存，仅开启 cache
	CacheModeFollow  = "follow"  // 追随后端（尊重 Cache-Control/Expires）
	CacheModeForce   = "force"   // 强制缓存（忽略后端头，TTL 由控制端下发）
	CacheModeBypass  = "bypass"  // 不缓存（强制回源，不落盘）
)
