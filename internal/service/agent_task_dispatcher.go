package service

import (
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/model"
	"gorm.io/gorm"
)

// AgentTaskDispatcher agent_tasks 派发服务
type AgentTaskDispatcher struct {
	db *gorm.DB
}

// NewAgentTaskDispatcher 创建派发服务
func NewAgentTaskDispatcher(db *gorm.DB) *AgentTaskDispatcher {
	return &AgentTaskDispatcher{db: db}
}

// DispatchResult 派发结果
type DispatchResult struct {
		TaskCreated         bool
		DispatchTriggered   bool
		TargetNodeCount     int
		AgentTaskCountAfter int
		Created             int
		Skipped             int
		Failed              int
		ErrorMsg            string
	}

// EnsureDispatched 确保 release_task 已派发到所有目标节点（幂等 + 补发）
func (d *AgentTaskDispatcher) EnsureDispatched(releaseTaskID int64, websiteID int64, traceID string) (*DispatchResult, error) {
	result := &DispatchResult{}

	// 1. 查询 release_task
	var releaseTask model.ReleaseTask
	if err := d.db.First(&releaseTask, releaseTaskID).Error; err != nil {
		return nil, fmt.Errorf("failed to query release_task: %w", err)
	}

	// 2. 只处理 targetType=website 的任务
	if releaseTask.TargetType != "website" {
		return nil, fmt.Errorf("unsupported targetType: %s", releaseTask.TargetType)
	}

	// 3. 查询 website
	var website model.Website
	if err := d.db.First(&website, websiteID).Error; err != nil {
		return nil, fmt.Errorf("failed to query website: %w", err)
	}

	// 4. 获取目标节点
	targetNodes, err := d.getTargetNodes(website.LineGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target nodes: %w", err)
	}

	result.TargetNodeCount = len(targetNodes)

	// 5. 无目标节点时标记失败（断链止血）
	if result.TargetNodeCount == 0 {
		errMsg := fmt.Sprintf("no eligible nodes for lineGroupId=%d", website.LineGroupID)
		result.ErrorMsg = errMsg
		result.DispatchTriggered = false
		d.db.Model(&releaseTask).Updates(map[string]interface{}{
			"status":     model.ReleaseTaskStatusFailed,
			"last_error": errMsg,
		})
		// 不返回错误，让上层接口返回 code=0
		return result, nil
	}

	// 6. 标记派发已触发
	result.DispatchTriggered = true

	// 7. 查询网站域名
	var domains []string
	var websiteDomains []model.WebsiteDomain
	if err := d.db.Where("website_id = ?", websiteID).Find(&websiteDomains).Error; err != nil {
		log.Printf("[Dispatcher] Failed to query website_domains: websiteId=%d, error=%v", websiteID, err)
	} else {
		for _, wd := range websiteDomains {
			domains = append(domains, wd.Domain)
		}
	}

	// 域名必填校验
	if len(domains) == 0 {
		errMsg := fmt.Sprintf("no domains found for websiteId=%d", websiteID)
		result.ErrorMsg = errMsg
		result.DispatchTriggered = false
		d.db.Model(&releaseTask).Updates(map[string]interface{}{
			"status":     model.ReleaseTaskStatusFailed,
			"last_error": errMsg,
		})
		return result, nil
	}

	// 8. 查询 origin_set_items 获取 origins（初始化为空数组而不是 nil）
	origins := make([]map[string]interface{}, 0)
	if website.OriginSetID.Valid && website.OriginSetID.Int32 > 0 {
		var originSetItems []model.OriginSetItem
		if err := d.db.Where("origin_set_id = ?", website.OriginSetID.Int32).Find(&originSetItems).Error; err != nil {
			log.Printf("[Dispatcher] Failed to query origin_set_items: originSetId=%d, error=%v", website.OriginSetID.Int32, err)
		} else {
			for _, item := range originSetItems {
				var snapshotData map[string]interface{}
				if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshotData); err != nil {
					log.Printf("[Dispatcher] Failed to unmarshal snapshot_json: id=%d, error=%v", item.ID, err)
					continue
				}
				// 提取 addresses 数组
				if addresses, ok := snapshotData["addresses"].([]interface{}); ok {
					for _, addr := range addresses {
						if addrMap, ok := addr.(map[string]interface{}); ok {
							// 只保留稳定字段
							origin := map[string]interface{}{
								"address":  addrMap["address"],
								"role":     addrMap["role"],
								"weight":   addrMap["weight"],
								"enabled":  addrMap["enabled"],
								"protocol": addrMap["protocol"],
							}
							origins = append(origins, origin)
						}
					}
				}
			}
		}
	}

	// 9. group 模式下 origins 必填校验
	if website.OriginMode == "group" && len(origins) == 0 {
		errMsg := fmt.Sprintf("no origins found for group mode: websiteId=%d, originSetId=%d", websiteID, website.OriginSetID.Int32)
		result.ErrorMsg = errMsg
		result.DispatchTriggered = false
		d.db.Model(&releaseTask).Updates(map[string]interface{}{
			"status":     model.ReleaseTaskStatusFailed,
			"last_error": errMsg,
		})
		return result, nil
	}

	// 10. 为每个节点创建 agent_task（幂等）
	for _, nodeID := range targetNodes {
		idKey := fmt.Sprintf("release-%d-node-%d", releaseTaskID, nodeID)

			// 幂等性检查：按 idKey 查询是否已存在（使用 JSON_EXTRACT）
			var count int64
			err := d.db.Model(&model.AgentTask{}).
				Where("node_id = ?", nodeID).
				Where("type = ?", model.TaskTypeApplyConfig).
				Where("JSON_UNQUOTE(JSON_EXTRACT(payload, '$.idKey')) = ?", idKey).
				Count(&count).Error

			if err != nil {
				result.Failed++
				result.ErrorMsg = fmt.Sprintf("failed to check existing task: %v", err)
				log.Printf("[Dispatcher] Failed to check existing task: nodeId=%d, error=%v", nodeID, err)
				continue
			}

			if count > 0 {
				// 已存在，跳过
				result.Skipped++
				log.Printf("[Dispatcher] agent_task already exists: idKey=%s", idKey)
				continue
			}

			// 构建 payload（根据 originMode 构建不同结构）
			payload := map[string]interface{}{
				"idKey":         idKey,
				"releaseTaskId": releaseTaskID,
				"websiteId":     websiteID,
				"lineGroupId":   website.LineGroupID,
				"originMode":    website.OriginMode,
				"type":          "applyConfig",
				"traceId":       traceID,
				"reload":        true,
				"domains":       domains,
			}

			// 根据 originMode 添加特定字段
			switch website.OriginMode {
			case "group":
				if website.OriginGroupID.Valid {
					payload["originGroupId"] = website.OriginGroupID.Int32
				}
				if website.OriginSetID.Valid {
					payload["originSetId"] = website.OriginSetID.Int32
				}
				payload["origins"] = map[string]interface{}{
					"items": origins,
				}
			case "redirect":
				payload["redirectUrl"] = website.RedirectURL
				payload["redirectStatusCode"] = website.RedirectStatusCode
			case "manual":
				if website.OriginSetID.Valid {
					payload["originSetId"] = website.OriginSetID.Int32
				}
				payload["origins"] = map[string]interface{}{
					"items": origins,
				}
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				result.Failed++
				result.ErrorMsg = fmt.Sprintf("failed to marshal payload: %v", err)
				continue
			}

			// 创建 agent_task
			agentTask := model.AgentTask{
				NodeID:  uint(nodeID),
				Type:    model.TaskTypeApplyConfig,
				Payload: string(payloadBytes),
				Status:  model.TaskStatusPending,
			}

			if err := d.db.Create(&agentTask).Error; err != nil {
				result.Failed++
				result.ErrorMsg = fmt.Sprintf("failed to create agent_task: %v", err)
				log.Printf("[Dispatcher] Failed to create agent_task: nodeId=%d, error=%v", nodeID, err)
				continue
			}

			result.Created++
			log.Printf("[Dispatcher] Created agent_task: idKey=%s, nodeId=%d", idKey, nodeID)
		}

	// 9. 统计当前 agent_tasks 数量（使用 JSON_EXTRACT 精确匹配）
	var agentTaskCount int64
	d.db.Model(&model.AgentTask{}).
		Where("JSON_EXTRACT(payload, '$.releaseTaskId') = ?", releaseTaskID).
		Count(&agentTaskCount)
	result.AgentTaskCountAfter = int(agentTaskCount)

	// 10. 更新 release_task.totalNodes
	d.db.Model(&releaseTask).Update("total_nodes", result.TargetNodeCount)

	// 11. 如果有失败，标记 release_task 失败
	if result.Failed > 0 {
		d.db.Model(&releaseTask).Updates(map[string]interface{}{
			"status":     model.ReleaseTaskStatusFailed,
			"last_error": result.ErrorMsg,
		})
		return result, fmt.Errorf("dispatch failed: %s", result.ErrorMsg)
	}

	log.Printf("[Dispatcher] EnsureDispatched completed: releaseTaskId=%d, targetNodes=%d, created=%d, skipped=%d, agentTaskCountAfter=%d",
		releaseTaskID, result.TargetNodeCount, result.Created, result.Skipped, result.AgentTaskCountAfter)

	return result, nil
}

// getTargetNodes 获取目标节点列表（去重）
func (d *AgentTaskDispatcher) getTargetNodes(lineGroupID int) ([]int, error) {
	// 查询链路：website.lineGroupId -> line_groups.node_group_id -> node_group_ips.ip_id -> node_ips.node_id -> nodes
	var nodeIDs []int

	// 构建查询
	query := d.db.Table("line_groups").
		Select("DISTINCT nodes.id").
		Joins("JOIN node_group_ips ON line_groups.node_group_id = node_group_ips.node_group_id").
		Joins("JOIN node_ips ON node_group_ips.ip_id = node_ips.id").
		Joins("JOIN nodes ON node_ips.node_id = nodes.id").
		Where("line_groups.id = ?", lineGroupID).
		Where("nodes.enabled = ?", true).
		Where("nodes.status = ?", "online").
		Where("node_ips.enabled = ?", true).
		Where("node_ips.status = ?", "active")

	// 打印执行的 SQL
	sqlStr := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Pluck("nodes.id", &nodeIDs)
	})
	log.Printf("[Dispatcher] getTargetNodes SQL: %s", sqlStr)

	// 执行查询
	err := query.Pluck("nodes.id", &nodeIDs).Error

	if err != nil {
		log.Printf("[Dispatcher] getTargetNodes error: lineGroupId=%d, error=%v", lineGroupID, err)
		return nil, err
	}

	log.Printf("[Dispatcher] getTargetNodes result: lineGroupId=%d, nodeIds=%v, count=%d", lineGroupID, nodeIDs, len(nodeIDs))

	return nodeIDs, nil
}
