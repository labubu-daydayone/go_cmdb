package service

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"go_cmdb/internal/model"
	"go_cmdb/internal/util"

	"gorm.io/gorm"
)

// HashPayload 用于计算 contentHash 的 payload 结构
type HashPayload struct {
	Origins []model.OriginItem `json:"origins"`
}

// ReleaseTaskService 发布任务服务
type ReleaseTaskService struct {
	db *gorm.DB
}

// NewReleaseTaskService 创建发布任务服务
func NewReleaseTaskService(db *gorm.DB) *ReleaseTaskService {
	return &ReleaseTaskService{db: db}
}

// CreateWebsiteReleaseTask 创建 Website 发布任务（带幂等性检查）
func (s *ReleaseTaskService) CreateWebsiteReleaseTask(websiteID int64, oldOriginSetID sql.NullInt32, traceID string) (*model.ReleaseTask, error) {
	// 查询 website
	var website model.Website
	if err := s.db.Preload("Domains").First(&website, websiteID).Error; err != nil {
		return nil, fmt.Errorf("failed to query website: %w", err)
	}

	// 只处理 group 模式的 website
	if website.OriginMode != model.OriginModeGroup {
		return nil, fmt.Errorf("website origin mode is not group")
	}

	// 必须有 origin_set_id
	if !website.OriginSetID.Valid {
		return nil, fmt.Errorf("website origin_set_id is not valid")
	}

	// 不再检查 originSetId 是否变化，由 contentHash 判断幂等

	originSetID := int64(website.OriginSetID.Int32)

	// 查询 origin_set_items
	var originSetItems []model.OriginSetItem
	if err := s.db.Where("origin_set_id = ?", originSetID).Find(&originSetItems).Error; err != nil {
		return nil, fmt.Errorf("failed to query origin_set_items: %w", err)
	}

	// 构建 origins.items
	originItems := make([]model.OriginItem, 0)
	for _, item := range originSetItems {
		// 解析 snapshot_json
		var snapshot struct {
			Addresses []struct {
				Address  string `json:"address"`
				Weight   int    `json:"weight"`
				Enabled  bool   `json:"enabled"`
				Role     string `json:"role"`
				Protocol string `json:"protocol"`
			} `json:"addresses"`
		}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err != nil {
			continue
		}

		for _, addr := range snapshot.Addresses {
			originItems = append(originItems, model.OriginItem{
				Address:  addr.Address,
				Role:     addr.Role,
				Weight:   addr.Weight,
				Enabled:  addr.Enabled,
				Protocol: addr.Protocol,
			})
		}
	}

	// 对 originItems 进行排序，确保相同地址生成相同 hash
	sort.Slice(originItems, func(i, j int) bool {
		if originItems[i].Role != originItems[j].Role {
			return originItems[i].Role < originItems[j].Role
		}
		if originItems[i].Protocol != originItems[j].Protocol {
			return originItems[i].Protocol < originItems[j].Protocol
		}
		if originItems[i].Address != originItems[j].Address {
			return originItems[i].Address < originItems[j].Address
		}
		return originItems[i].Weight < originItems[j].Weight
	})

	// 获取 originGroupId（如果有）
	var originGroupID int64
	if website.OriginGroupID.Valid {
		originGroupID = int64(website.OriginGroupID.Int32)
	}

	// 构建用于 hash 计算的 payload（不包含 originSetId）
	hashPayload := HashPayload{
		Origins: originItems,
	}

	// 构建完整的 payload（包含 originSetId，用于存储）
	payload := model.ReleaseTaskPayload{
		// 旧字段（保留兼容）
		WebsiteID: int(websiteID),

		// 新字段
		TargetType:    "website",
		TargetID:      websiteID,
		OriginSetID:   originSetID,
		OriginGroupID: originGroupID,
		OriginMode:    string(website.OriginMode),
		LineGroupID:   int64(website.LineGroupID),
		Origins: &model.OriginsList{
			Items: originItems,
		},
	}

	// 序列化 hashPayload 为 JSON（用于计算 hash）
	// 由于使用结构体，字段顺序已经稳定
	stablePayloadBytes, err := json.Marshal(hashPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 校验：origins.items 不能为空
	if len(originItems) == 0 {
		return nil, fmt.Errorf("origin_set has no valid addresses")
	}

	// 计算 contentHash = sha256(stableJson(payload))，基于渲染内容
	hash := sha256.Sum256(stablePayloadBytes)
	contentHash := hex.EncodeToString(hash[:])

	// 打点3：hash 计算完成
	util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=hash_input_ready bytesLen=%d sha256=%s", 
		traceID, len(stablePayloadBytes), contentHash)

	// 写入文件用于对比
	debugDir := "/opt/go_cmdb/var/debug"
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=hash_file_written path=%s ok=0 error=%v", 
			traceID, debugDir, err)
	} else {
		filePath := fmt.Sprintf("%s/hashpayload_%d_%d.json", debugDir, websiteID, originSetID)
		if err := os.WriteFile(filePath, stablePayloadBytes, 0644); err != nil {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=hash_file_written path=%s ok=0 error=%v", 
				traceID, filePath, err)
		} else {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=hash_file_written path=%s ok=1", 
				traceID, filePath)
		}
	}

	// 幂等性检查：相同 contentHash 的 pending/running 任务
	var existingTask model.ReleaseTask
	err = s.db.Where("target_type = ? AND target_id = ? AND content_hash = ? AND status IN (?, ?)",
		"website", websiteID, contentHash, model.ReleaseTaskStatusPending, model.ReleaseTaskStatusRunning).
		First(&existingTask).Error

	if err == nil {
		// 已存在相同任务，补发 agent_tasks
		util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=reuse_existing_task releaseTaskId=%d",
			traceID, existingTask.ID)
		dispatcher := NewAgentTaskDispatcher(s.db)
		if _, err := dispatcher.EnsureDispatched(existingTask.ID, websiteID, traceID); err != nil {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_failed releaseTaskId=%d error=%v",
				traceID, existingTask.ID, err)
		} else {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_success releaseTaskId=%d",
				traceID, existingTask.ID)
		}
		return &existingTask, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing task: %w", err)
	}

	// 创建新任务
	task := model.ReleaseTask{
		Type:        model.ReleaseTaskTypeApplyConfig,
		Status:      model.ReleaseTaskStatusPending,
		TargetType:  "website",
		TargetID:    websiteID,
		ContentHash: contentHash,
		Payload:     &payload,
		RetryCount:  0,
	}

	if err := s.db.Create(&task).Error; err != nil {
		return nil, fmt.Errorf("failed to create release task: %w", err)
	}

	// 派发到 agent_tasks
	dispatcher := NewAgentTaskDispatcher(s.db)
	if _, err := dispatcher.EnsureDispatched(task.ID, websiteID, traceID); err != nil {
		// dispatch 失败已经写入 lastError，这里只记录日志
		util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_failed releaseTaskId=%d error=%v",
			traceID, task.ID, err)
	} else {
		util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_success releaseTaskId=%d",
			traceID, task.ID)
	}

	return &task, nil
}

// CreateWebsiteReleaseTaskWithDispatch 创建 release_task 并返回 dispatch 结果
func (s *ReleaseTaskService) CreateWebsiteReleaseTaskWithDispatch(websiteID int64, oldOriginSetID *int32, traceID string) (*model.ReleaseTask, *DispatchResult, error) {
		dispatchResult := &DispatchResult{}

	// 获取 website
	var website model.Website
	if err := s.db.First(&website, websiteID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query website: %w", err)
	}

	// 只处理 group 模式
	if website.OriginMode != model.OriginModeGroup {
		return nil, nil, fmt.Errorf("website originMode is not group")
	}

	if !website.OriginSetID.Valid {
		return nil, nil, fmt.Errorf("website originSetId is null")
	}

	originSetID := int64(website.OriginSetID.Int32)

	// 获取 origin_set
	var originSet model.OriginSet
	if err := s.db.First(&originSet, originSetID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query origin_set: %w", err)
	}

	// 获取 origin_set_items
	var originSetItems []model.OriginSetItem
	if err := s.db.Where("origin_set_id = ?", originSet.ID).Find(&originSetItems).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query origin_set_items: %w", err)
	}

	// 构建 originItems
	originItems := make([]model.OriginItem, 0, len(originSetItems))
	for _, item := range originSetItems {
		var snapshot struct {
			Addresses []struct {
				Address  string `json:"address"`
				Weight   int    `json:"weight"`
				Enabled  bool   `json:"enabled"`
				Role     string `json:"role"`
				Protocol string `json:"protocol"`
			} `json:"addresses"`
		}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err != nil {
			continue
		}
		for _, addr := range snapshot.Addresses {
			originItems = append(originItems, model.OriginItem{
				Address:  addr.Address,
				Role:     addr.Role,
				Weight:   addr.Weight,
				Enabled:  addr.Enabled,
				Protocol: addr.Protocol,
			})
		}
	}

	// 排序
	sort.Slice(originItems, func(i, j int) bool {
		if originItems[i].Address != originItems[j].Address {
			return originItems[i].Address < originItems[j].Address
		}
		if originItems[i].Role != originItems[j].Role {
			return originItems[i].Role < originItems[j].Role
		}
		if originItems[i].Protocol != originItems[j].Protocol {
			return originItems[i].Protocol < originItems[j].Protocol
		}
		return originItems[i].Weight < originItems[j].Weight
	})

	// 构建用于 hash 计算的 payload（不包含 originSetId）
	hashPayload := HashPayload{
		Origins: originItems,
	}

	// 计算 contentHash
	payloadBytes, err := json.Marshal(hashPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	hash := sha256.Sum256(payloadBytes)
	contentHash := hex.EncodeToString(hash[:])

	util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=hash_input_ready bytesLen=%d sha256=%s",
		traceID, len(payloadBytes), contentHash)

	// 构建完整的 payload（包含 originSetId，用于存储）
	var originGroupID int64
	if website.OriginGroupID.Valid {
		originGroupID = int64(website.OriginGroupID.Int32)
	}
	payload := model.ReleaseTaskPayload{
		WebsiteID:     int(websiteID),
		TargetType:    "website",
		TargetID:      websiteID,
		OriginSetID:   originSetID,
		OriginGroupID: originGroupID,
		OriginMode:    string(website.OriginMode),
		LineGroupID:   int64(website.LineGroupID),
		Origins: &model.OriginsList{
			Items: originItems,
		},
	}

	// 幂等性检查
	var existingTask model.ReleaseTask
	err = s.db.Where("target_type = ? AND target_id = ? AND content_hash = ? AND status IN (?, ?)",
		"website", websiteID, contentHash, model.ReleaseTaskStatusPending, model.ReleaseTaskStatusRunning).
		First(&existingTask).Error

if err == nil {
			// 已存在相同任务，补发 agent_tasks
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=reuse_existing_task releaseTaskId=%d",
				traceID, existingTask.ID)
			dispatcher := NewAgentTaskDispatcher(s.db)
			result, err := dispatcher.EnsureDispatched(existingTask.ID, websiteID, traceID)
			if err != nil {
				util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_failed releaseTaskId=%d error=%v",
					traceID, existingTask.ID, err)
			} else {
				util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_success releaseTaskId=%d",
					traceID, existingTask.ID)
				dispatchResult = result
				dispatchResult.TaskCreated = false // Reusing task
				dispatchResult.DispatchTriggered = true
			}
			return &existingTask, dispatchResult, nil
		}

	if err != gorm.ErrRecordNotFound {
		return nil, nil, fmt.Errorf("failed to check existing task: %w", err)
	}

	// 创建新任务
	task := model.ReleaseTask{
		Type:        model.ReleaseTaskTypeApplyConfig,
		Status:      model.ReleaseTaskStatusPending,
		TargetType:  "website",
		TargetID:    websiteID,
		ContentHash: contentHash,
		Payload:     &payload,
		RetryCount:  0,
	}

if err := s.db.Create(&task).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to create release task: %w", err)
		}
		dispatchResult.TaskCreated = true

	// 派发到 agent_tasks
	dispatcher := NewAgentTaskDispatcher(s.db)
result, err := dispatcher.EnsureDispatched(task.ID, websiteID, traceID)
		if err != nil {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_failed releaseTaskId=%d error=%v",
				traceID, task.ID, err)
		} else {
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=dispatch_success releaseTaskId=%d",
				traceID, task.ID)
			dispatchResult.DispatchTriggered = true
			dispatchResult.TargetNodeCount = result.TargetNodeCount
			dispatchResult.AgentTaskCountAfter = result.AgentTaskCountAfter
			dispatchResult.Created = result.Created
			dispatchResult.Skipped = result.Skipped
			dispatchResult.Failed = result.Failed
			dispatchResult.ErrorMsg = result.ErrorMsg
		}

	return &task, dispatchResult, nil
}

// stableJSONMarshal 生成稳定的 JSON（字段排序）
func stableJSONMarshal(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	// 解析为 map 并重新排序
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", err
	}

	// 排序 keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 重新构建 JSON
	result := "{"
	for i, k := range keys {
		if i > 0 {
			result += ","
		}
		keyJSON, _ := json.Marshal(k)
		valueJSON, _ := json.Marshal(m[k])
		result += string(keyJSON) + ":" + string(valueJSON)
	}
	result += "}"

	return result, nil
}
