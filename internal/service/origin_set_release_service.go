package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"go_cmdb/internal/model"
	"gorm.io/gorm"
)

// OriginSetReleaseService origin_set 发布服务
type OriginSetReleaseService struct {
	db *gorm.DB
}

// NewOriginSetReleaseService 创建 origin_set 发布服务实例
func NewOriginSetReleaseService(db *gorm.DB) *OriginSetReleaseService {
	return &OriginSetReleaseService{
		db: db,
	}
}

// OriginSetReleaseResult origin_set 发布任务创建结果
type OriginSetReleaseResult struct {
	ReleaseTaskID int64
	TaskCreated   bool
	SkipReason    string
}

// CreateOriginSetReleaseTask 创建 origin_set 发布任务（upstream）
func (s *OriginSetReleaseService) CreateOriginSetReleaseTask(originSetID int64, traceID string) (*OriginSetReleaseResult, error) {
	result := &OriginSetReleaseResult{}

	// 1. 查询 origin_set_items
	var originSetItems []model.OriginSetItem
	if err := s.db.Where("origin_set_id = ?", originSetID).Find(&originSetItems).Error; err != nil {
		return nil, fmt.Errorf("failed to query origin_set_items: %w", err)
	}

	// 2. 解析 origins
	type OriginItem struct {
		Address  string `json:"address"`
		Role     string `json:"role"`
		Weight   int    `json:"weight"`
		Enabled  bool   `json:"enabled"`
		Protocol string `json:"protocol"`
	}

	origins := make([]OriginItem, 0)
	var originGroupID int64

	for _, item := range originSetItems {
		var snapshotData map[string]interface{}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshotData); err != nil {
			log.Printf("[OriginSetReleaseService] Failed to unmarshal snapshot_json: id=%d, error=%v", item.ID, err)
			continue
		}

		// 提取 originGroupId
		if ogid, ok := snapshotData["originGroupId"].(float64); ok {
			originGroupID = int64(ogid)
		}

		// 提取 addresses 数组
		if addresses, ok := snapshotData["addresses"].([]interface{}); ok {
			for _, addr := range addresses {
				if addrMap, ok := addr.(map[string]interface{}); ok {
					origin := OriginItem{
						Address:  addrMap["address"].(string),
						Role:     addrMap["role"].(string),
						Enabled:  addrMap["enabled"].(bool),
						Protocol: addrMap["protocol"].(string),
					}
					if weight, ok := addrMap["weight"].(float64); ok {
						origin.Weight = int(weight)
					}
					origins = append(origins, origin)
				}
			}
		}
	}

	// 3. 排序 origins（确保 content_hash 稳定）
	sort.Slice(origins, func(i, j int) bool {
		if origins[i].Role != origins[j].Role {
			return origins[i].Role < origins[j].Role
		}
		if origins[i].Address != origins[j].Address {
			return origins[i].Address < origins[j].Address
		}
		return origins[i].Weight < origins[j].Weight
	})

	// 4. 构建 content_hash（只依赖 originSetId 和 origins）
	type ContentHashPayload struct {
		OriginSetID int64        `json:"originSetId"`
		Origins     []OriginItem `json:"origins"`
	}

	hashPayload := ContentHashPayload{
		OriginSetID: originSetID,
		Origins:     origins,
	}

	contentJSON, _ := json.Marshal(hashPayload)
	hashBytes := sha256.Sum256(contentJSON)
	contentHash := hex.EncodeToString(hashBytes[:])

	// 5. 查询是否存在相同 content_hash 的任务
	var existingTask model.ReleaseTask
	err := s.db.Where("target_type = ? AND target_id = ? AND content_hash = ?",
		"origin_set", originSetID, contentHash).
		Order("id DESC").
		First(&existingTask).Error

	if err == nil {
		// 找到相同 content_hash 的任务，跳过创建
		log.Printf("[OriginSetReleaseService] Skip creating release_task: originSetId=%d, existingTaskId=%d, reason=same_content_hash", originSetID, existingTask.ID)
		result.ReleaseTaskID = existingTask.ID
		result.TaskCreated = false
		result.SkipReason = "same_content_hash"
		return result, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query existing release_task: %w", err)
	}

	// 6. 创建新的 release_task（upstream）
	releaseTask := model.ReleaseTask{
		Type:        "apply_config",
		TargetType:  "origin_set",
		TargetID:    originSetID,
		Status:      model.ReleaseTaskStatusPending,
		ContentHash: contentHash,
		RetryCount:  0,
	}

	if err := s.db.Create(&releaseTask).Error; err != nil {
		return nil, fmt.Errorf("failed to create release_task: %w", err)
	}

	log.Printf("[OriginSetReleaseService] Created upstream release_task: id=%d, originSetId=%d, originGroupId=%d, contentHash=%s, traceId=%s",
		releaseTask.ID, originSetID, originGroupID, contentHash, traceID)

	result.ReleaseTaskID = releaseTask.ID
	result.TaskCreated = true

	return result, nil
}
