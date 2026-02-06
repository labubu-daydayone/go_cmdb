package upstream

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"time"

	"gorm.io/gorm"
)

// Publisher upstream 发布服务
type Publisher struct {
	db       *gorm.DB
	renderer *Renderer
	selector *NodeSelector
}

// NewPublisher 创建发布服务
func NewPublisher(db *gorm.DB) *Publisher {
	return &Publisher{
		db:       db,
		renderer: NewRenderer(db),
		selector: NewNodeSelector(db),
	}
}

// PublishRequest 发布请求
type PublishRequest struct {
	WebsiteID   int64 `json:"websiteId"`
	OriginSetID int64 `json:"originSetId"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	ReleaseID int64 `json:"releaseId"`
	TaskID    int64 `json:"taskId"` // 返回第一个创建的 agent_task id
}

// FileItem 文件项
type FileItem struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

// FilesPayload files 对象
type FilesPayload struct {
	Items []FileItem `json:"items"`
}

// ApplyConfigPayload apply_config 任务的 payload 结构
type ApplyConfigPayload struct {
	ConfigHash string       `json:"configHash"`
	Files      FilesPayload `json:"files"`
	Reload     bool         `json:"reload"`
}

// Publish 发布 upstream 配置到节点
func (p *Publisher) Publish(req *PublishRequest) (*PublishResponse, error) {
	var releaseID int64
	var firstTaskID int64

	err := p.db.Transaction(func(tx *gorm.DB) error {
		// 1. 渲染 upstream 配置
		renderResp, err := p.renderer.Render(&RenderRequest{
			OriginSetID: req.OriginSetID,
			WebsiteID:   req.WebsiteID,
		})
		if err != nil {
			return err
		}

		// 2. 计算 configHash
		configHash := fmt.Sprintf("%x", sha256.Sum256([]byte(renderResp.UpstreamConf)))

		// 3. 查询最近 50 条 apply_config 任务，检查幂等性
		var recentTasks []model.AgentTask
		err = tx.Where("type = ?", model.TaskTypeApplyConfig).
			Order("created_at DESC").
			Limit(50).
			Find(&recentTasks).Error
		if err != nil {
			return httpx.ErrDatabaseError("failed to query recent tasks", err)
		}

		// 4. 检查是否有相同 website 且 configHash 一致的任务
		targetPath := fmt.Sprintf("/data/vhost/upstream/upstream_website_%d.conf", req.WebsiteID)
		isIdempotent := false
		for _, task := range recentTasks {
			var payload ApplyConfigPayload
			if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
				continue // 跳过无法解析的任务
			}

			// 检查是否包含相同的文件路径
			for _, file := range payload.Files.Items {
				if file.Path == targetPath {
					// 找到了同一个 website 的任务，比较 hash
					if payload.ConfigHash == configHash {
						isIdempotent = true
						break
					}
				}
			}

			if isIdempotent {
				break
			}
		}

		// 5. 如果幂等，直接返回
		if isIdempotent {
			return nil // 不创建任务，releaseID 和 taskID 保持为 0
		}

		// 6. 选择目标节点
		nodeIDs, err := p.selector.SelectNodesForWebsite(req.WebsiteID)
		if err != nil {
			return err
		}
		if len(nodeIDs) == 0 {
			return httpx.ErrStateConflict("no available nodes")
		}

		// 7. 构造 payload
		payload := ApplyConfigPayload{
			ConfigHash: configHash,
			Files: FilesPayload{
				Items: []FileItem{
					{
						Path:    targetPath,
						Content: renderResp.UpstreamConf,
						Mode:    "0644",
					},
				},
			},
			Reload: true,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return httpx.ErrInternalError("failed to marshal payload", err)
		}

		// 8. 创建 agent_tasks（apply_config）
		for _, nodeID := range nodeIDs {
			// 生成 requestID
			requestID := fmt.Sprintf("apply_%d_%d_%d", req.WebsiteID, nodeID, time.Now().UnixNano())

			// 创建任务
			task := &model.AgentTask{
				NodeID:    uint(nodeID),
				Type:      model.TaskTypeApplyConfig,
				Payload:   string(payloadJSON),
				Status:    model.TaskStatusPending,
				RequestID: requestID,
			}
			if err := tx.Create(task).Error; err != nil {
				return httpx.ErrDatabaseError("failed to create apply_config task", err)
			}

			// 记录第一个任务 ID
			if firstTaskID == 0 {
				firstTaskID = int64(task.ID)
			}
		}

		// 9. 创建 reload 任务（带去抖控制）
		for _, nodeID := range nodeIDs {
			// 检查 10 秒内是否已有 pending/running 的 reload 任务
			var count int64
			err := tx.Model(&model.AgentTask{}).
				Where("node_id = ? AND type = ? AND status IN ? AND created_at > ?",
					nodeID, model.TaskTypeReload, []string{model.TaskStatusPending, model.TaskStatusRunning}, time.Now().Add(-10*time.Second)).
				Count(&count).Error
			if err != nil {
				return httpx.ErrDatabaseError("failed to check reload tasks", err)
			}

			// 如果已有，跳过
			if count > 0 {
				continue
			}

			// 创建 reload 任务
			requestID := fmt.Sprintf("reload_%d_%d_%d", req.WebsiteID, nodeID, time.Now().UnixNano())
			task := &model.AgentTask{
				NodeID:    uint(nodeID),
				Type:      model.TaskTypeReload,
				Payload:   "{}",
				Status:    model.TaskStatusPending,
				RequestID: requestID,
			}
			if err := tx.Create(task).Error; err != nil {
				return httpx.ErrDatabaseError("failed to create reload task", err)
			}
		}

		// 10. 创建 release_task（用于前端查询状态）
		// 生成 version
		var maxVersion int64
		err = tx.Model(&model.ReleaseTask{}).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error
		if err != nil {
			return httpx.ErrDatabaseError("failed to generate version", err)
		}
		version := maxVersion + 1

		// 创建 release_task
		releaseTask := &model.ReleaseTask{
			Type:       model.ReleaseTaskTypeApplyConfig,
			Target:     model.ReleaseTaskTargetCDN,
			Version:    version,
			Status:     model.ReleaseTaskStatusPending,
			TotalNodes: len(nodeIDs),
		}
		if err := tx.Create(releaseTask).Error; err != nil {
			return httpx.ErrDatabaseError("failed to create release task", err)
		}

		releaseID = releaseTask.ID

		// 创建 release_task_nodes
		for i, nodeID := range nodeIDs {
			node := &model.ReleaseTaskNode{
				ReleaseTaskID: releaseTask.ID,
				NodeID:        nodeID,
				Batch:         (i / 10) + 1, // 每批 10 个节点
				Status:        model.ReleaseTaskNodeStatusPending,
			}
			if err := tx.Create(node).Error; err != nil {
				return httpx.ErrDatabaseError("failed to create release task node", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PublishResponse{
		ReleaseID: releaseID,
		TaskID:    firstTaskID,
	}, nil
}
