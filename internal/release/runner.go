package release

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go_cmdb/internal/agentclient"
	"go_cmdb/internal/model"
	"gorm.io/gorm"
)

var (
	releaseTaskNodesTableChecked bool
	releaseTaskNodesTableExists  bool
	tableCheckMutex              sync.Mutex
)

// Runner 单个release_task的执行状态机
type Runner struct {
	db          *gorm.DB
	agentClient *agentclient.Client
	task        *model.ReleaseTask
}

// NewRunner 创建Runner
func NewRunner(db *gorm.DB, agentClient *agentclient.Client, task *model.ReleaseTask) *Runner {
	return &Runner{
		db:          db,
		agentClient: agentClient,
		task:        task,
	}
}

// Run 执行发布任务
func (r *Runner) Run() error {
	log.Printf("[Runner] Start running release task %d (version=%d)", r.task.ID, r.task.Version)

	// 获取所有batch（按batch升序）
	batches, err := r.getAllBatches()
	if err != nil {
		return fmt.Errorf("failed to get batches: %w", err)
	}

	log.Printf("[Runner] Found %d batches for release task %d", len(batches), r.task.ID)

	// If no batches found (e.g., table does not exist), skip execution
	if len(batches) == 0 {
		log.Printf("[Runner] No batches found for release task %d. Skipping execution (likely due to missing release_task_nodes table).", r.task.ID)
		return nil
	}

	// 按batch顺序执行
	for _, batch := range batches {
		log.Printf("[Runner] Processing batch %d for release task %d", batch, r.task.ID)

		// 处理当前batch
		if err := r.processBatch(batch); err != nil {
			log.Printf("[Runner] Batch %d failed: %v", batch, err)
			// 标记发布任务失败
			r.handleFailure()
			return fmt.Errorf("batch %d failed: %w", batch, err)
		}

		log.Printf("[Runner] Batch %d completed successfully", batch)
	}

	// 所有batch成功，标记发布任务成功
	r.handleSuccess()
	log.Printf("[Runner] Release task %d completed successfully", r.task.ID)

	return nil
}

// getAllBatches 获取所有batch（去重并排序）
func (r *Runner) getAllBatches() ([]int, error) {
	// Check if release_task_nodes table exists (only once)
	tableCheckMutex.Lock()
	if !releaseTaskNodesTableChecked {
		releaseTaskNodesTableChecked = true
		var count int64
		err := r.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'release_task_nodes'").Scan(&count).Error
		if err != nil {
			log.Printf("[Runner] Warning: Failed to check if release_task_nodes table exists: %v. Assuming table does not exist and degrading gracefully.", err)
			releaseTaskNodesTableExists = false
		} else {
			releaseTaskNodesTableExists = (count > 0)
			if !releaseTaskNodesTableExists {
				log.Printf("[Runner] Warning: Table 'release_task_nodes' does not exist. Degrading to skip release_task_nodes-dependent logic. This warning will only be printed once.")
			}
		}
	}
	tableCheckMutex.Unlock()

	// If table does not exist, return empty batches to skip the logic
	if !releaseTaskNodesTableExists {
		return []int{}, nil
	}

	var batches []int
	err := r.db.Model(&model.ReleaseTaskNode{}).
		Where("release_task_id = ?", r.task.ID).
		Distinct("batch").
		Order("batch ASC").
		Pluck("batch", &batches).Error

	return batches, err
}

// processBatch 处理单个batch
func (r *Runner) processBatch(batch int) error {
	// 获取当前batch的所有nodes
	var nodes []model.ReleaseTaskNode
	if err := r.db.Where("release_task_id = ? AND batch = ?", r.task.ID, batch).
		Order("node_id ASC").
		Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	log.Printf("[Runner] Batch %d has %d nodes", batch, len(nodes))

	// 第一轮：dispatch所有pending节点
	for i := range nodes {
		if nodes[i].Status == model.ReleaseTaskNodeStatusPending {
			if err := r.dispatchNode(&nodes[i]); err != nil {
				// dispatch失败，标记节点为failed
				r.markNodeFailed(&nodes[i], err.Error())
				return fmt.Errorf("failed to dispatch node %d: %w", nodes[i].NodeID, err)
			}
		}
	}

	// 第二轮：轮询所有running节点直到完成
	maxPolls := 120 // 最多轮询120次（10分钟，每5秒一次）
	pollInterval := 5 * time.Second

	for poll := 0; poll < maxPolls; poll++ {
		// 重新加载nodes状态
		if err := r.db.Where("release_task_id = ? AND batch = ?", r.task.ID, batch).
			Order("node_id ASC").
			Find(&nodes).Error; err != nil {
			return fmt.Errorf("failed to reload nodes: %w", err)
		}

		// 检查是否所有节点都已完成
		allCompleted := true
		hasFailure := false

		for i := range nodes {
			if nodes[i].Status == model.ReleaseTaskNodeStatusRunning {
				allCompleted = false
				// 查询节点状态
				if err := r.pollNode(&nodes[i]); err != nil {
					log.Printf("[Runner] Failed to poll node %d: %v", nodes[i].NodeID, err)
					// 查询失败，标记节点为failed
					r.markNodeFailed(&nodes[i], err.Error())
					hasFailure = true
					break
				}
			} else if nodes[i].Status == model.ReleaseTaskNodeStatusFailed {
				hasFailure = true
			}
		}

		// 如果有失败，立即返回
		if hasFailure {
			return fmt.Errorf("batch %d has failed nodes", batch)
		}

		// 如果所有节点都已完成，退出轮询
		if allCompleted {
			log.Printf("[Runner] Batch %d all nodes completed", batch)
			break
		}

		// 等待下一次轮询
		time.Sleep(pollInterval)
	}

	// 最终检查：确保所有节点都是success状态
	for i := range nodes {
		if nodes[i].Status != model.ReleaseTaskNodeStatusSuccess {
			return fmt.Errorf("node %d is not in success status (current: %s)", nodes[i].NodeID, nodes[i].Status)
		}
	}

	return nil
}

// dispatchNode dispatch单个节点
func (r *Runner) dispatchNode(node *model.ReleaseTaskNode) error {
	// 获取节点信息
	var n model.Node
	if err := r.db.First(&n, node.NodeID).Error; err != nil {
		return fmt.Errorf("failed to get node info: %w", err)
	}

	log.Printf("[Runner] Dispatching node %d (IP=%s)", node.NodeID, n.MainIP)

	// 调用Agent dispatch接口
	taskID, err := r.agentClient.Dispatch(n.MainIP, r.task.Version)
	if err != nil {
		return fmt.Errorf("failed to dispatch to agent: %w", err)
	}

	log.Printf("[Runner] Node %d dispatched successfully (taskID=%s)", node.NodeID, taskID)

	// 更新节点状态为running
	now := time.Now()
	if err := r.db.Model(node).Updates(map[string]interface{}{
		"status":     model.ReleaseTaskNodeStatusRunning,
		"started_at": &now,
	}).Error; err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}

	return nil
}

// pollNode 轮询单个节点状态
func (r *Runner) pollNode(node *model.ReleaseTaskNode) error {
	// 获取节点信息
	var n model.Node
	if err := r.db.First(&n, node.NodeID).Error; err != nil {
		return fmt.Errorf("failed to get node info: %w", err)
	}

	// 生成taskID（与dispatch时一致）
	taskID := fmt.Sprintf("apply_config_%s_%d", n.MainIP, r.task.Version)

	// 调用Agent query接口
	status, lastError, err := r.agentClient.Query(n.MainIP, taskID)
	if err != nil {
		return fmt.Errorf("failed to query agent: %w", err)
	}

	log.Printf("[Runner] Node %d status: %s (lastError=%s)", node.NodeID, status, lastError)

	// 根据状态更新节点
	switch status {
	case "success":
		r.markNodeSuccess(node)
	case "failed":
		r.markNodeFailed(node, lastError)
	case "pending", "running":
		// 继续等待
	default:
		return fmt.Errorf("unknown status: %s", status)
	}

	return nil
}

// markNodeSuccess 标记节点成功
func (r *Runner) markNodeSuccess(node *model.ReleaseTaskNode) {
	now := time.Now()
	if err := r.db.Model(node).Updates(map[string]interface{}{
		"status":      model.ReleaseTaskNodeStatusSuccess,
		"finished_at": &now,
	}).Error; err != nil {
		log.Printf("[Runner] Failed to mark node %d as success: %v", node.NodeID, err)
	}

	// 更新release_tasks的success_nodes计数
	if err := r.db.Model(&model.ReleaseTask{}).
		Where("id = ?", r.task.ID).
		UpdateColumn("success_nodes", gorm.Expr("success_nodes + 1")).Error; err != nil {
		log.Printf("[Runner] Failed to update success_nodes: %v", err)
	}
}

// markNodeFailed 标记节点失败
func (r *Runner) markNodeFailed(node *model.ReleaseTaskNode, errorMsg string) {
	now := time.Now()
	if err := r.db.Model(node).Updates(map[string]interface{}{
		"status":      model.ReleaseTaskNodeStatusFailed,
		"error_msg":   &errorMsg,
		"finished_at": &now,
	}).Error; err != nil {
		log.Printf("[Runner] Failed to mark node %d as failed: %v", node.NodeID, err)
	}

	// 更新release_tasks的failed_nodes计数
	if err := r.db.Model(&model.ReleaseTask{}).
		Where("id = ?", r.task.ID).
		UpdateColumn("failed_nodes", gorm.Expr("failed_nodes + 1")).Error; err != nil {
		log.Printf("[Runner] Failed to update failed_nodes: %v", err)
	}
}

// handleFailure 处理发布失败
func (r *Runner) handleFailure() {
	log.Printf("[Runner] Handling failure for release task %d", r.task.ID)

	// 1. 更新release_tasks.status = failed
	if err := r.db.Model(&model.ReleaseTask{}).
		Where("id = ?", r.task.ID).
		Update("status", model.ReleaseTaskStatusFailed).Error; err != nil {
		log.Printf("[Runner] Failed to update task status: %v", err)
	}

	// 2. 标记后续batch的nodes为skipped
	// 获取当前最大batch（已处理的batch）
	var currentBatch int
	if err := r.db.Model(&model.ReleaseTaskNode{}).
		Where("release_task_id = ? AND status IN ?", r.task.ID, []string{
			string(model.ReleaseTaskNodeStatusRunning),
			string(model.ReleaseTaskNodeStatusSuccess),
			string(model.ReleaseTaskNodeStatusFailed),
		}).
		Select("MAX(batch)").
		Scan(&currentBatch).Error; err != nil {
		log.Printf("[Runner] Failed to get current batch: %v", err)
		return
	}

	// 标记后续batch的pending节点为skipped
	if err := r.db.Model(&model.ReleaseTaskNode{}).
		Where("release_task_id = ? AND batch > ? AND status = ?",
			r.task.ID, currentBatch, model.ReleaseTaskNodeStatusPending).
		Update("status", model.ReleaseTaskNodeStatusSkipped).Error; err != nil {
		log.Printf("[Runner] Failed to mark nodes as skipped: %v", err)
	}

	log.Printf("[Runner] Marked subsequent batches as skipped")
}

// handleSuccess 处理发布成功
func (r *Runner) handleSuccess() {
	log.Printf("[Runner] Handling success for release task %d", r.task.ID)

	// 更新release_tasks.status = success
	if err := r.db.Model(&model.ReleaseTask{}).
		Where("id = ?", r.task.ID).
		Update("status", model.ReleaseTaskStatusSuccess).Error; err != nil {
		log.Printf("[Runner] Failed to update task status: %v", err)
	}
}
