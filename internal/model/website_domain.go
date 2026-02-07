package model

// WebsiteDomain 网站域名
type WebsiteDomain struct {
	BaseModel
	WebsiteID int    `gorm:"not null;index" json:"website_id"`
	Domain    string `gorm:"type:varchar(255);uniqueIndex;not null" json:"domain"` // 域名必须唯一
	IsPrimary bool   `gorm:"type:tinyint;default:0" json:"is_primary"`             // 是否主域名
	CNAME     string `gorm:"type:varchar(255)" json:"cname"`                       // CNAME 指向地址

	// 关联
	Website *Website `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
}

// TableName 指定表名
func (WebsiteDomain) TableName() string {
	return "website_domains"
}
