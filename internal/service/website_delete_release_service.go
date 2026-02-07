package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go_cmdb/internal/model"
	"log"
	"sort"

	"gorm.io/gorm"
)

// WebsiteDeleteInfo 网站删除信息
type WebsiteDeleteInfo struct {
	WebsiteID   int64
	LineGroupID int64
	Domains     []string
}

// CreateWebsiteDeleteReleaseTask 创建网站删除发布任务
func CreateWebsiteDeleteReleaseTask(db *gorm.DB, info *WebsiteDeleteInfo, traceID string) (int64, error) {
	// 1. 构建 payload
	payload := map[string]interface{}{
		"type":        "applyConfig",
		"action":      "delete",
		"websiteId":   info.WebsiteID,
		"lineGroupId": info.LineGroupID,
		"domains":     info.Domains,
		"delete": map[string]interface{}{
			"serverFiles":   []string{fmt.Sprintf("/data/vhost/server/server_%d.conf", info.WebsiteID)},
			"upstreamFiles": []string{fmt.Sprintf("/data/vhost/upstream/upstream_%d.conf", info.WebsiteID)},
		},
		"reload":  true,
		"traceId": traceID,
	}

	// 2. 计算 content_hash（action + websiteId + fileList）
	hashData := map[string]interface{}{
		"action":    "delete",
		"websiteId": info.WebsiteID,
		"serverFiles": []string{fmt.Sprintf("/data/vhost/server/server_%d.conf", info.WebsiteID)},
		"upstreamFiles": []string{fmt.Sprintf("/data/vhost/upstream/upstream_%d.conf", info.WebsiteID)},
	}

	// 排序 domains 以确保幂等性
	sortedDomains := make([]string, len(info.Domains))
	copy(sortedDomains, info.Domains)
	sort.Strings(sortedDomains)
	hashData["domains"] = sortedDomains

	hashBytes, _ := json.Marshal(hashData)
	contentHash := fmt.Sprintf("%x", sha256.Sum256(hashBytes))

	// 3. 幂等性检查：查询是否已存在相同 content_hash 的任务
	var existingTask model.ReleaseTask
	err := db.Where("target_type = ? AND target_id = ? AND content_hash = ?",
		"website", info.WebsiteID, contentHash).
		First(&existingTask).Error

	if err == nil {
		// 已存在，返回已有任务 ID
		log.Printf("[WebsiteDeleteRelease] Task already exists: releaseTaskId=%d, contentHash=%s", existingTask.ID, contentHash)
		return int64(existingTask.ID), nil
	}

	if err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("failed to check existing task: %w", err)
	}

	// 4. 创建新的 release_task
	payloadBytes, _ := json.Marshal(payload)
	payloadObj := &model.ReleaseTaskPayload{}
	payloadObj.Scan(payloadBytes)

	releaseTask := &model.ReleaseTask{
		TargetType:  "website",
		TargetID:    info.WebsiteID,
		Type:        model.ReleaseTaskTypeApplyConfig,
		Status:      model.ReleaseTaskStatusPending,
		Payload:     payloadObj,
		ContentHash: contentHash,
		TotalNodes:  0, // 初始为 0，派发时更新
	}

	if err := db.Create(releaseTask).Error; err != nil {
		return 0, fmt.Errorf("failed to create release_task: %w", err)
	}

	log.Printf("[WebsiteDeleteRelease] Created release_task: id=%d, websiteId=%d, contentHash=%s",
		releaseTask.ID, info.WebsiteID, contentHash)

	return int64(releaseTask.ID), nil
}
