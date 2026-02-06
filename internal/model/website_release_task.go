package model

import "time"

// WebsiteReleaseTaskStatus 网站发布任务状态
type WebsiteReleaseTaskStatus string

const (
	WebsiteReleaseTaskStatusPending   WebsiteReleaseTaskStatus = "pending"
	WebsiteReleaseTaskStatusRunning   WebsiteReleaseTaskStatus = "running"
	WebsiteReleaseTaskStatusSuccess   WebsiteReleaseTaskStatus = "success"
	WebsiteReleaseTaskStatusFailed    WebsiteReleaseTaskStatus = "failed"
	WebsiteReleaseTaskStatusCancelled WebsiteReleaseTaskStatus = "cancelled"
)

// WebsiteReleaseTask 网站发布任务
type WebsiteReleaseTask struct {
	ID           int64                    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WebsiteID    int64                    `gorm:"column:website_id;not null;index:idx_website_release_tasks_website_id" json:"websiteId"`
	Status       WebsiteReleaseTaskStatus `gorm:"column:status;type:enum('pending','running','success','failed','cancelled');not null;default:pending;index:idx_website_release_tasks_status" json:"status"`
	ContentHash  string                   `gorm:"column:content_hash;type:char(64);not null;uniqueIndex:uniq_website_release_tasks_website_hash" json:"contentHash"`
	Payload      string                   `gorm:"column:payload;type:longtext;not null" json:"payload"`
	ErrorMessage *string                  `gorm:"column:error_message;type:varchar(2048)" json:"errorMessage"`
	CreatedAt    time.Time                `gorm:"column:created_at;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt    time.Time                `gorm:"column:updated_at;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 指定表名
func (WebsiteReleaseTask) TableName() string {
	return "website_release_tasks"
}
