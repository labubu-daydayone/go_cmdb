package model

// OriginSet 回源快照
type OriginSet struct {
	BaseModel
	Name          string `gorm:"type:varchar(255);not null" json:"name"`
	Description   string `gorm:"type:text" json:"description"`
	Status        string `gorm:"type:varchar(32);not null;default:active" json:"status"`
	Source        string `gorm:"type:enum('group','manual');not null" json:"source"`
	OriginGroupID int64  `gorm:"default:0;not null" json:"originGroupId"`

	// 关联
	Items       []OriginSetItem `gorm:"foreignKey:OriginSetID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	OriginGroup *OriginGroup    `gorm:"foreignKey:OriginGroupID" json:"originGroup,omitempty"`
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
