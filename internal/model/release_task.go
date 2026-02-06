package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

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

// ReleaseTaskPayload 发布任务 payload
type ReleaseTaskPayload struct {
	// 旧字段（保留兼容）
	WebsiteID int                            `json:"websiteId"`
	Files     map[string]ReleaseTaskFileInfo `json:"files,omitempty"`

	// 新增字段（website 发布任务）
	TargetType    string        `json:"targetType,omitempty"`
	TargetID      int64         `json:"targetId,omitempty"`
	OriginSetID   int64         `json:"originSetId,omitempty"`
	OriginGroupID int64         `json:"originGroupId,omitempty"`
	OriginMode    string        `json:"originMode,omitempty"`
	LineGroupID   int64         `json:"lineGroupId,omitempty"`
	Origins       *OriginsList  `json:"origins,omitempty"`
}

// OriginsList origins 列表
type OriginsList struct {
	Items []OriginItem `json:"items"`
}

// OriginItem origin 项
type OriginItem struct {
	Address  string `json:"address"`
	Role     string `json:"role"`
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
	Protocol string `json:"protocol"`
}

// ReleaseTaskFileInfo 文件信息
type ReleaseTaskFileInfo struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Value 实现 driver.Valuer 接口
func (p ReleaseTaskPayload) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan 实现 sql.Scanner 接口
func (p *ReleaseTaskPayload) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, p)
}

// ReleaseTask 发布任务
type ReleaseTask struct {
	// 旧字段（保留兼容）
	ID           int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Type         ReleaseTaskType   `gorm:"column:type;type:varchar(64);not null;default:'apply_config'" json:"type"`
	Target       ReleaseTaskTarget `gorm:"column:target;type:varchar(32);not null;default:''" json:"target"`
	Version      int64             `gorm:"column:version;type:bigint;not null;default:1" json:"version"`
	Status       ReleaseTaskStatus `gorm:"column:status;type:varchar(32);not null" json:"status"`
	TotalNodes   int               `gorm:"column:total_nodes;type:int;not null;default:0" json:"totalNodes"`
	SuccessNodes int               `gorm:"column:success_nodes;type:int;not null;default:0" json:"successNodes"`
	FailedNodes  int               `gorm:"column:failed_nodes;type:int;not null;default:0" json:"failedNodes"`

	// 新增字段（website 发布任务）
	TargetType  string              `gorm:"type:varchar(32);not null;default:'website'" json:"targetType"`
	TargetID    int64               `gorm:"type:bigint;not null;default:0" json:"targetId"`
	ContentHash string              `gorm:"type:char(64);not null;default:''" json:"contentHash"`
	Payload     *ReleaseTaskPayload `gorm:"type:longtext" json:"payload"`
	LastError   *string             `gorm:"type:text" json:"lastError"`
	RetryCount  int                 `gorm:"type:int;not null;default:0" json:"retryCount"`
	NextRetryAt *time.Time          `gorm:"type:datetime(3)" json:"nextRetryAt"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updatedAt"`
}

// TableName 指定表名
func (ReleaseTask) TableName() string {
	return "release_tasks"
}
