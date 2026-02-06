package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/model"
	"gorm.io/gorm"
)

// WebsiteReleaseService 网站发布服务
type WebsiteReleaseService struct {
	db         *gorm.DB
	dispatcher *AgentTaskDispatcher
}

// NewWebsiteReleaseService 创建网站发布服务实例
func NewWebsiteReleaseService(db *gorm.DB) *WebsiteReleaseService {
	return &WebsiteReleaseService{
		db:         db,
		dispatcher: NewAgentTaskDispatcher(db),
	}
}

// CreateReleaseTaskResult 创建发布任务结果
type CreateReleaseTaskResult struct {
	ReleaseTaskID          int64
	TaskCreated            bool
	SkipReason             string
	DispatchTriggered      bool
	TargetNodeCount        int
	CreatedAgentTaskCount  int
	SkippedAgentTaskCount  int
	AgentTaskCountAfter    int
	PayloadValid           bool
	PayloadInvalidReason   string
}

// CreateWebsiteReleaseTaskWithDispatch 创建网站发布任务并派发到 Agent
func (s *WebsiteReleaseService) CreateWebsiteReleaseTaskWithDispatch(websiteID int64, traceID string) (*CreateReleaseTaskResult, error) {
	result := &CreateReleaseTaskResult{}

	// 1. 查询 website
	var website model.Website
	if err := s.db.Preload("Domains").First(&website, websiteID).Error; err != nil {
		return nil, fmt.Errorf("failed to query website: %w", err)
	}

	// 2. 构建 content_hash
	contentData := map[string]interface{}{
		"websiteId":     website.ID,
		"lineGroupId":   website.LineGroupID,
		"cacheRuleId":   website.CacheRuleID,
		"originMode":    website.OriginMode,
		"originGroupId": website.OriginGroupID.Int32,
		"originSetId":   website.OriginSetID.Int32,
		"redirectUrl":   website.RedirectURL,
		"status":        website.Status,
	}
	contentJSON, _ := json.Marshal(contentData)
	hashBytes := sha256.Sum256(contentJSON)
	contentHash := hex.EncodeToString(hashBytes[:])

	// 3. 查询是否存在相同 content_hash 的任务
	var existingTask model.ReleaseTask
	err := s.db.Where("target_type = ? AND target_id = ? AND content_hash = ?",
		"website", websiteID, contentHash).
		Order("id DESC").
		First(&existingTask).Error

	if err == nil {
		// 找到相同 content_hash 的任务，跳过创建
		log.Printf("[WebsiteReleaseService] Skip creating release_task: websiteId=%d, existingTaskId=%d, reason=same_content_hash", websiteID, existingTask.ID)
		result.ReleaseTaskID = existingTask.ID
		result.TaskCreated = false
		result.SkipReason = "same_content_hash"
		return result, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query existing release_task: %w", err)
	}

	// 4. 创建新的 release_task
	releaseTask := model.ReleaseTask{
		Type:        "apply_config",
		TargetType:  "website",
		TargetID:    websiteID,
		Status:      model.ReleaseTaskStatusPending,
		ContentHash: contentHash,
		RetryCount:  0,
	}

	if err := s.db.Create(&releaseTask).Error; err != nil {
		return nil, fmt.Errorf("failed to create release_task: %w", err)
	}

	log.Printf("[WebsiteReleaseService] Created release_task: id=%d, websiteId=%d, contentHash=%s", releaseTask.ID, websiteID, contentHash)

	result.ReleaseTaskID = releaseTask.ID
	result.TaskCreated = true

	// 5. 派发到 Agent
	dispatchResult, err := s.dispatcher.EnsureDispatched(releaseTask.ID, websiteID, traceID)
	if err != nil {
		log.Printf("[WebsiteReleaseService] Failed to dispatch release_task: id=%d, error=%v", releaseTask.ID, err)
		return nil, fmt.Errorf("failed to dispatch release_task: %w", err)
	}

	result.DispatchTriggered = dispatchResult.DispatchTriggered
	result.TargetNodeCount = dispatchResult.TargetNodeCount
	result.CreatedAgentTaskCount = dispatchResult.Created
	result.SkippedAgentTaskCount = dispatchResult.Skipped
	result.AgentTaskCountAfter = dispatchResult.AgentTaskCountAfter

	// 设置 payload 有效性
	if dispatchResult.ErrorMsg != "" {
		result.PayloadValid = false
		result.PayloadInvalidReason = dispatchResult.ErrorMsg
	} else {
		result.PayloadValid = true
	}

	log.Printf("[WebsiteReleaseService] Dispatch completed: releaseTaskId=%d, targetNodeCount=%d, created=%d, skipped=%d",
		releaseTask.ID, result.TargetNodeCount, result.CreatedAgentTaskCount, result.SkippedAgentTaskCount)

	return result, nil
}
