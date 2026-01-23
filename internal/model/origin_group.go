package model

// OriginGroup 回源分组（可复用）
type OriginGroup struct {
	BaseModel
	Name        string `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:varchar(255)" json:"description"`
	Status      string `gorm:"type:enum('active','inactive');default:'active'" json:"status"`

	// 关联
	Addresses []OriginGroupAddress `gorm:"foreignKey:OriginGroupID;constraint:OnDelete:CASCADE" json:"addresses,omitempty"`
}

// TableName 指定表名
func (OriginGroup) TableName() string {
	return "origin_groups"
}

// Status constants
const (
	OriginGroupStatusActive   = "active"
	OriginGroupStatusInactive = "inactive"
)
