package model

import "database/sql"

// Website 网站配置
type Website struct {
	BaseModel
	Domain      string `gorm:"type:varchar(255);uniqueIndex:uk_websites_domain" json:"domain"` // 域名（唯一标识）
	LineGroupID int    `gorm:"not null;index" json:"lineGroupId"`                              // 线路分组ID
	CacheRuleID int `gorm:"default:0;index" json:"cacheRuleId"` // 缓存规则ID（可选）

	// 回源配置
	OriginMode         string        `gorm:"type:enum('group','manual','redirect');not null" json:"originMode"`
	OriginGroupID      sql.NullInt32 `gorm:"index;default:null" json:"originGroupId"`      // group模式时有值，其他为 NULL
	OriginSetID        sql.NullInt32 `gorm:"index;default:null" json:"originSetId"`        // 绑定 origin set 时有值，否则为 NULL
	RedirectURL        string        `gorm:"type:varchar(2048)" json:"redirectUrl"`        // redirect模式时有值
	RedirectStatusCode int           `gorm:"default:0" json:"redirectStatusCode"`          // redirect模式时有值

	Status string `gorm:"type:enum('active','inactive');default:'active'" json:"status"`

	// 关联
	LineGroup   *LineGroup   `gorm:"foreignKey:LineGroupID" json:"line_group,omitempty"`
	OriginGroup *OriginGroup `gorm:"foreignKey:OriginGroupID" json:"origin_group,omitempty"`
	OriginSet   *OriginSet   `gorm:"foreignKey:OriginSetID" json:"origin_set,omitempty"`
	Domains     []WebsiteDomain `gorm:"foreignKey:WebsiteID" json:"domains,omitempty"`
	HTTPS       *WebsiteHTTPS   `gorm:"foreignKey:WebsiteID" json:"https,omitempty"`
}

// TableName 指定表名
func (Website) TableName() string {
	return "websites"
}

// OriginMode constants
const (
	OriginModeGroup    = "group"
	OriginModeManual   = "manual"
	OriginModeRedirect = "redirect"
)

// Status constants
const (
	WebsiteStatusActive   = "active"
	WebsiteStatusInactive = "inactive"
)
