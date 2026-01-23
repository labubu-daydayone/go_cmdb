package model

import "time"

// ReleaseTaskNodeStatus 发布任务节点状态
type ReleaseTaskNodeStatus string

const (
	ReleaseTaskNodeStatusPending ReleaseTaskNodeStatus = "pending"
	ReleaseTaskNodeStatusRunning ReleaseTaskNodeStatus = "running"
	ReleaseTaskNodeStatusSuccess ReleaseTaskNodeStatus = "success"
	ReleaseTaskNodeStatusFailed  ReleaseTaskNodeStatus = "failed"
	ReleaseTaskNodeStatusSkipped ReleaseTaskNodeStatus = "skipped"
)

// ReleaseTaskNode 发布任务节点
type ReleaseTaskNode struct {
	ID            int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	ReleaseTaskID int64                 `gorm:"not null;index:idx_release_task_id;uniqueIndex:uk_release_task_node" json:"release_task_id"`
	NodeID        int                   `gorm:"not null;index:idx_node_id;uniqueIndex:uk_release_task_node" json:"node_id"`
	Batch         int                   `gorm:"not null;default:1;index:idx_batch" json:"batch"`
	Status        ReleaseTaskNodeStatus `gorm:"type:enum('pending','running','success','failed','skipped');not null;default:pending;index:idx_status" json:"status"`
	ErrorMsg      *string               `gorm:"type:varchar(255)" json:"error_msg"`
	StartedAt     *time.Time            `json:"started_at"`
	FinishedAt    *time.Time            `json:"finished_at"`
	CreatedAt     time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ReleaseTaskNode) TableName() string {
	return "release_task_nodes"
}
