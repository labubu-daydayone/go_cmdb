package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go_cmdb/internal/model"
	"strings"

	"gorm.io/gorm"
)

// WebsiteReleaseTaskService 网站发布任务服务
type WebsiteReleaseTaskService struct {
	db *gorm.DB
}

// NewWebsiteReleaseTaskService 创建网站发布任务服务
func NewWebsiteReleaseTaskService(db *gorm.DB) *WebsiteReleaseTaskService {
	return &WebsiteReleaseTaskService{db: db}
}

// RenderPayload 渲染网站配置 payload
func (s *WebsiteReleaseTaskService) RenderPayload(websiteID int64) (string, error) {
	// 查询网站信息
	var website model.Website
	if err := s.db.Preload("Domains").Preload("OriginSet").First(&website, websiteID).Error; err != nil {
		return "", fmt.Errorf("failed to query website: %w", err)
	}

	// 构建 payload 结构
	payload := map[string]interface{}{
		"websiteId":   website.ID,
		"lineGroupId": website.LineGroupID,
		"originMode":  website.OriginMode,
		"domains":     []string{},
	}

	// 添加域名列表
	domains := make([]string, 0, len(website.Domains))
	for _, d := range website.Domains {
		domains = append(domains, d.Domain)
	}
	payload["domains"] = domains

	// 根据 originMode 添加不同的配置
	switch website.OriginMode {
	case model.OriginModeGroup:
		if website.OriginSetID.Valid {
			// 查询 origin_set_items
			var items []model.OriginSetItem
			if err := s.db.Where("origin_set_id = ?", website.OriginSetID.Int32).Find(&items).Error; err != nil {
				return "", fmt.Errorf("failed to query origin_set_items: %w", err)
			}

			// 解析 snapshot_json 并构建 upstream 配置
			upstreams := make([]map[string]interface{}, 0)
			for _, item := range items {
				var snapshot map[string]interface{}
				if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err != nil {
					continue
				}
				if addresses, ok := snapshot["addresses"].([]interface{}); ok {
					for _, addr := range addresses {
						if addrMap, ok := addr.(map[string]interface{}); ok {
							upstreams = append(upstreams, map[string]interface{}{
								"ip":     addrMap["ip"],
								"port":   addrMap["port"],
								"weight": addrMap["weight"],
							})
						}
					}
				}
			}
			payload["upstreams"] = upstreams
		}

	case model.OriginModeRedirect:
		payload["redirectUrl"] = website.RedirectURL
		payload["redirectStatusCode"] = website.RedirectStatusCode
	}

	// 序列化为 JSON 字符串
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	return string(payloadBytes), nil
}

// CalculateContentHash 计算 payload 的 SHA256 哈希
func (s *WebsiteReleaseTaskService) CalculateContentHash(payload string) string {
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}

// CreateOrUpdateTask 创建或更新网站发布任务（带幂等性）
func (s *WebsiteReleaseTaskService) CreateOrUpdateTask(websiteID int64) (*model.WebsiteReleaseTask, error) {
	// 渲染 payload
	payload, err := s.RenderPayload(websiteID)
	if err != nil {
		return nil, err
	}

	// 计算 contentHash
	contentHash := s.CalculateContentHash(payload)

	// 幂等性检查：查询是否已存在相同 contentHash 且状态为 pending/running/success 的任务
	var existingTask model.WebsiteReleaseTask
	err = s.db.Where("website_id = ? AND content_hash = ? AND status IN (?, ?, ?)",
		websiteID, contentHash,
		model.WebsiteReleaseTaskStatusPending,
		model.WebsiteReleaseTaskStatusRunning,
		model.WebsiteReleaseTaskStatusSuccess).
		Order("id DESC").
		First(&existingTask).Error

	if err == nil {
		// 已存在相同任务，直接返回
		return &existingTask, nil
	}

	if err != gorm.ErrRecordNotFound {
		// 查询出错
		return nil, fmt.Errorf("failed to query existing task: %w", err)
	}

	// 检查是否有 failed/cancelled 的任务可以复用
	var failedTask model.WebsiteReleaseTask
	err = s.db.Where("website_id = ? AND content_hash = ? AND status IN (?, ?)",
		websiteID, contentHash,
		model.WebsiteReleaseTaskStatusFailed,
		model.WebsiteReleaseTaskStatusCancelled).
		Order("id DESC").
		First(&failedTask).Error

	if err == nil {
		// 复用 failed/cancelled 任务，重置为 pending
		failedTask.Status = model.WebsiteReleaseTaskStatusPending
		failedTask.ErrorMessage = nil
		failedTask.Payload = payload // 更新 payload
		if err := s.db.Save(&failedTask).Error; err != nil {
			return nil, fmt.Errorf("failed to reset failed task: %w", err)
		}
		return &failedTask, nil
	}

	// 创建新任务
	task := &model.WebsiteReleaseTask{
		WebsiteID:   websiteID,
		Status:      model.WebsiteReleaseTaskStatusPending,
		ContentHash: contentHash,
		Payload:     payload,
	}

	if err := s.db.Create(task).Error; err != nil {
		// 检查是否是唯一索引冲突
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "uniq_website_release_tasks_website_hash") {
			// 并发情况下可能已经创建，重新查询
			if err := s.db.Where("website_id = ? AND content_hash = ?", websiteID, contentHash).
				Order("id DESC").
				First(&existingTask).Error; err == nil {
				return &existingTask, nil
			}
		}
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// RetryTask 重试失败的任务
func (s *WebsiteReleaseTaskService) RetryTask(taskID int64) error {
	var task model.WebsiteReleaseTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// 只允许 failed/cancelled 状态的任务重试
	if task.Status != model.WebsiteReleaseTaskStatusFailed && task.Status != model.WebsiteReleaseTaskStatusCancelled {
		return fmt.Errorf("only failed or cancelled tasks can be retried")
	}

	// 重置状态
	task.Status = model.WebsiteReleaseTaskStatusPending
	task.ErrorMessage = nil

	if err := s.db.Save(&task).Error; err != nil {
		return fmt.Errorf("failed to retry task: %w", err)
	}

	return nil
}
