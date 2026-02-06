package model

// WebsiteHTTPS HTTPS配置
type WebsiteHTTPS struct {
	BaseModel
	WebsiteID     int    `gorm:"uniqueIndex;not null" json:"website_id"` // 一个网站只有一个HTTPS配置
	Enabled       bool   `gorm:"type:tinyint;default:0" json:"enabled"`
	ForceRedirect bool   `gorm:"type:tinyint;default:0" json:"force_redirect"` // 强制HTTPS重定向
	HSTS          bool   `gorm:"type:tinyint;default:0" json:"hsts"`           // HTTP Strict Transport Security
	CertMode      string `gorm:"type:enum('select','acme');default:'select'" json:"cert_mode"`

	// select模式
	CertificateID *uint `gorm:"column:certificate_id;index" json:"certificate_id"` // 选择已有证书

	// acme模式
	ACMEProviderID *uint `gorm:"column:acme_provider_id;index" json:"acme_provider_id"` // ACME Provider ID
	ACMEAccountID  *uint `gorm:"column:acme_account_id;index" json:"acme_account_id"`  // ACME Account ID

	// 关联
	Website *Website `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
}

// TableName 指定表名
func (WebsiteHTTPS) TableName() string {
	return "website_https"
}

// CertMode constants
const (
	CertModeSelect = "select"
	CertModeACME   = "acme"
)
