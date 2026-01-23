package model

// OriginSet 网站回源快照（不可复用）
// 一个 origin_set 只能属于一个网站
type OriginSet struct {
	BaseModel
	Source        string `gorm:"type:enum('group','manual');not null" json:"source"`
	OriginGroupID int    `gorm:"default:0;not null" json:"origin_group_id"` // source=group时有值

	// 关联
	Addresses   []OriginAddress `gorm:"foreignKey:OriginSetID;constraint:OnDelete:CASCADE" json:"addresses,omitempty"`
	OriginGroup *OriginGroup    `gorm:"foreignKey:OriginGroupID" json:"origin_group,omitempty"`
}

// TableName 指定表名
func (OriginSet) TableName() string {
	return "origin_sets"
}

// Source constants
const (
	OriginSetSourceGroup  = "group"
	OriginSetSourceManual = "manual"
)
