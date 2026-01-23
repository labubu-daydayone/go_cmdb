package model

import "time"

// ReleaseTaskType 发布任务类型
type ReleaseTaskType string

const (
	ReleaseTaskTypeApplyConfig ReleaseTaskType = "apply_config"
)

// ReleaseTaskTarget 发布目标类型
type ReleaseTaskTarget string

const (
	ReleaseTaskTargetCDN ReleaseTaskTarget = "cdn"
)

// ReleaseTaskStatus 发布任务状态
type ReleaseTaskStatus string

const (
	ReleaseTaskStatusPending ReleaseTaskStatus = "pending"
	ReleaseTaskStatusRunning ReleaseTaskStatus = "running"
	ReleaseTaskStatusSuccess ReleaseTaskStatus = "success"
	ReleaseTaskStatusFailed  ReleaseTaskStatus = "failed"
	ReleaseTaskStatusPaused  ReleaseTaskStatus = "paused"
)

// ReleaseTask 发布任务
type ReleaseTask struct {
	ID           int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Type         ReleaseTaskType   `gorm:"type:enum('apply_config');not null" json:"type"`
	Target       ReleaseTaskTarget `gorm:"type:enum('cdn');not null" json:"target"`
	Version      int64             `gorm:"not null;uniqueIndex:uk_version" json:"version"`
	Status       ReleaseTaskStatus `gorm:"type:enum('pending','running','success','failed','paused');not null;default:pending" json:"status"`
	TotalNodes   int               `gorm:"not null;default:0" json:"total_nodes"`
	SuccessNodes int               `gorm:"not null;default:0" json:"success_nodes"`
	FailedNodes  int               `gorm:"not null;default:0" json:"failed_nodes"`
	CreatedAt    time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ReleaseTask) TableName() string {
	return "release_tasks"
}
