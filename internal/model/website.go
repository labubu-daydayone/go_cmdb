package model

// Website 网站配置
type Website struct {
	BaseModel
	Domain string `gorm:"type:varchar(255);uniqueIndex;not null" json:"domain"`
	Status string `gorm:"type:enum('active','inactive');default:'active'" json:"status"`

	// 回源配置
	OriginMode           string `gorm:"type:enum('group','manual','redirect');not null" json:"origin_mode"`
	OriginGroupID        int    `gorm:"default:0;not null" json:"origin_group_id"`        // group模式时有值
	OriginSetID          int    `gorm:"default:0;not null" json:"origin_set_id"`          // group/manual模式时有值
	RedirectURL          string `gorm:"type:varchar(512)" json:"redirect_url"`            // redirect模式时有值
	RedirectStatusCode   int    `gorm:"default:0" json:"redirect_status_code"`            // redirect模式时有值

	// 关联
	OriginGroup *OriginGroup `gorm:"foreignKey:OriginGroupID" json:"origin_group,omitempty"`
	OriginSet   *OriginSet   `gorm:"foreignKey:OriginSetID" json:"origin_set,omitempty"`
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
