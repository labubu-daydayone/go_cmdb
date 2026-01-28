package model

import "time"

// OriginSetItem 回源快照项
type OriginSetItem struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OriginSetID   int64     `gorm:"column:origin_set_id;not null;index" json:"originSetId"`
	OriginGroupID int64     `gorm:"column:origin_group_id;not null;index" json:"originGroupId"`
	SnapshotJSON  string    `gorm:"column:snapshot_json;type:json;not null" json:"snapshotJson"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (OriginSetItem) TableName() string {
	return "origin_set_items"
}
