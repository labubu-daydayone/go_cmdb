package service

import (
	"encoding/json"
	"fmt"
	"go_cmdb/internal/model"
	"log"
)

// EnsureDispatchedForDelete 专门用于删除场景的派发逻辑（不查询 website）
func (d *AgentTaskDispatcher) EnsureDispatchedForDelete(releaseTaskID int64, lineGroupID int64, domains []string, traceID string) (*DispatchResult, error) {
	result := &DispatchResult{}

	// 1. 查询 release_task
	var releaseTask model.ReleaseTask
	if err := d.db.First(&releaseTask, releaseTaskID).Error; err != nil {
		return nil, fmt.Errorf("failed to query release_task: %w", err)
	}

	// 2. 获取目标节点
	targetNodes, err := d.getTargetNodes(int(lineGroupID))
	if err != nil {
		return nil, fmt.Errorf("failed to get target nodes: %w", err)
	}

	result.TargetNodeCount = len(targetNodes)

	// 3. 无目标节点时不派发（但任务保持 pending，不标记为 failed）
	if result.TargetNodeCount == 0 {
		errMsg := "no eligible nodes, dispatch skipped"
		result.ErrorMsg = errMsg
		result.DispatchTriggered = false
		// 更新 last_error 但保持 status 为 pending（不改为 failed）
		d.db.Model(&releaseTask).Updates(map[string]interface{}{
			"total_nodes": 0,
			"last_error":  errMsg,
		})
		log.Printf("[Dispatcher] No eligible nodes for lineGroupId=%d, releaseTaskId=%d remains pending", lineGroupID, releaseTaskID)
		// 不返回错误，让上层接口返回 code=0
		return result, nil
	}

	// 4. 标记派发已触发
	result.DispatchTriggered = true

	// 5. 构建 payload（使用传入的 domains）
	payload := make(map[string]interface{})
	payload["releaseTaskId"] = releaseTaskID
	payload["action"] = "delete"
	payload["domains"] = domains
	payload["traceId"] = traceID

	// 6. 派发到每个目标节点
	for _, nodeID := range targetNodes {
		// 生成 idKey（用于幂等性检查）
		idKey := fmt.Sprintf("release-%d-node-%d", releaseTaskID, nodeID)

		// 幂等性检查：查询是否已存在
		var existingTask model.AgentTask
		err := d.db.Where("JSON_EXTRACT(payload, '$.idKey') = ?", idKey).First(&existingTask).Error
		if err == nil {
			// 已存在，跳过
			result.Skipped++
			log.Printf("[Dispatcher] agent_task already exists: idKey=%s", idKey)
			continue
		}

		// 添加 idKey 到 payload
		payload["idKey"] = idKey

		// 序列化 payload
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
			log.Printf("[Dispatcher] Failed to create agent_task: nodeId=%d, error=%v", nodeID, err)
			result.Failed++
			continue
		}

		result.Created++
		log.Printf("[Dispatcher] Created agent_task: id=%d, nodeId=%d, releaseTaskId=%d", agentTask.ID, nodeID, releaseTaskID)
	}

	// 7. 统计派发后的 agent_task 数量
	var agentTaskCount int64
	d.db.Model(&model.AgentTask{}).Where("JSON_EXTRACT(payload, '$.releaseTaskId') = ?", releaseTaskID).Count(&agentTaskCount)
	result.AgentTaskCountAfter = int(agentTaskCount)

	// 8. 更新 release_task 的 total_nodes
	d.db.Model(&releaseTask).Updates(map[string]interface{}{
		"total_nodes": result.TargetNodeCount,
	})

	log.Printf("[Dispatcher] EnsureDispatchedForDelete completed: releaseTaskId=%d, targetNodes=%d, created=%d, skipped=%d, agentTaskCountAfter=%d",
		releaseTaskID, result.TargetNodeCount, result.Created, result.Skipped, result.AgentTaskCountAfter)

	return result, nil
}
