package service

import (
	"go_cmdb/internal/model"
	"go_cmdb/internal/renderer"

	"gorm.io/gorm"
)

// ReleaseTaskService 发布任务服务
type ReleaseTaskService struct {
	db       *gorm.DB
	renderer *renderer.WebsiteConfigRenderer
}

// NewReleaseTaskService 创建发布任务服务
func NewReleaseTaskService(db *gorm.DB) *ReleaseTaskService {
	return &ReleaseTaskService{
		db:       db,
		renderer: renderer.NewWebsiteConfigRenderer(db),
	}
}

// CreateWebsiteReleaseTask 创建网站发布任务（带幂等性检查）
func (s *ReleaseTaskService) CreateWebsiteReleaseTask(websiteID int) (*model.ReleaseTask, error) {
	// 渲染配置
	payload, contentHash, err := s.renderer.RenderConfig(websiteID)
	if err != nil {
		return nil, err
	}

	// 幂等性检查：查询是否已存在相同 contentHash 且状态为 pending 或 running 的任务
	var existingTask model.ReleaseTask
	err = s.db.Where("target_type = ? AND target_id = ? AND content_hash = ? AND status IN (?, ?)",
		"website", websiteID, contentHash,
		model.ReleaseTaskStatusPending, model.ReleaseTaskStatusRunning).
		First(&existingTask).Error

	if err == nil {
		// 已存在相同任务，直接返回
		return &existingTask, nil
	}

	if err != gorm.ErrRecordNotFound {
		// 查询出错
		return nil, err
	}

	// 创建新任务
	task := &model.ReleaseTask{
		Type:        "apply_config",
		Status:      model.ReleaseTaskStatusPending,
		TargetType:  "website",
		TargetID:    int64(websiteID),
		ContentHash: contentHash,
		Payload:     payload,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	return task, nil
}
