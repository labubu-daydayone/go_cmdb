package release

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service 发布服务
type Service struct {
	db *gorm.DB
}

// NewService 创建发布服务
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// GenerateVersion 生成新版本号
// 使用MAX(version)+1策略
func (s *Service) GenerateVersion(tx *gorm.DB) (int64, error) {
	var maxVersion int64
	err := tx.Model(&model.ReleaseTask{}).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return 0, err
	}
	return maxVersion + 1, nil
}

// CreateRelease 创建发布任务
// 事务性创建：version生成 + release_tasks + release_task_nodes
func (s *Service) CreateRelease(req *CreateReleaseRequest) (*CreateReleaseResponse, error) {
	var resp *CreateReleaseResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 生成version
		version, err := s.GenerateVersion(tx)
		if err != nil {
			return err
		}

		// 2. 选择在线节点
		nodes, err := SelectOnlineNodes(tx)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return httpx.ErrStateConflict("no online nodes")
		}

		// 3. 分配批次
		batches := AllocateBatches(nodes)

		// 4. 创建release_tasks
		task := &model.ReleaseTask{
			Type:       model.ReleaseTaskTypeApplyConfig,
			Target:     model.ReleaseTaskTarget(req.Target),
			Version:    version,
			Status:     model.ReleaseTaskStatusPending,
			TotalNodes: len(nodes),
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}

		// 5. 批量创建release_task_nodes
		for _, batch := range batches {
			for _, nodeID := range batch.NodeIDs {
				node := &model.ReleaseTaskNode{
					ReleaseTaskID: task.ID,
					NodeID:        nodeID,
					Batch:         batch.Batch,
					Status:        model.ReleaseTaskNodeStatusPending,
				}
				if err := tx.Create(node).Error; err != nil {
					return err
				}
			}
		}

		// 6. 构造响应
		resp = &CreateReleaseResponse{
			ReleaseID:  task.ID,
			Version:    version,
			TotalNodes: len(nodes),
			Batches:    batches,
		}

		return nil
	})

	return resp, err
}

// GetReleaseResponse 获取发布任务响应
type GetReleaseResponse struct {
	ReleaseID    int64             `json:"releaseId"`
	Version      int64             `json:"version"`
	Status       string            `json:"status"`
	TotalNodes   int               `json:"totalNodes"`
	SuccessNodes int               `json:"successNodes"`
	FailedNodes  int               `json:"failedNodes"`
	Batches      []BatchNodeStatus `json:"batches"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

// BatchNodeStatus batch节点状态
type BatchNodeStatus struct {
	Batch int          `json:"batch"`
	Nodes []NodeStatus `json:"nodes"`
}

// NodeStatus 节点状态
type NodeStatus struct {
	NodeID     int     `json:"nodeId"`
	Status     string  `json:"status"`
	ErrorMsg   *string `json:"errorMsg,omitempty"`
	StartedAt  *string `json:"startedAt,omitempty"`
	FinishedAt *string `json:"finishedAt,omitempty"`
}

// GetRelease 获取发布任务详情
func (s *Service) GetRelease(releaseID int64) (*GetReleaseResponse, error) {
	// 查询release_task
	var task model.ReleaseTask
	if err := s.db.First(&task, releaseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.ErrNotFound("release task not found")
		}
		return nil, err
	}

	// 查询所有节点（按batch和node_id排序）
	var nodes []model.ReleaseTaskNode
	if err := s.db.Where("release_task_id = ?", releaseID).
		Order("batch ASC, node_id ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	// 按batch分组
	batchMap := make(map[int][]NodeStatus)
	for _, node := range nodes {
		nodeStatus := NodeStatus{
			NodeID:   node.NodeID,
			Status:   string(node.Status),
			ErrorMsg: node.ErrorMsg,
		}

		if node.StartedAt != nil {
			startedAt := node.StartedAt.Format("2006-01-02T15:04:05Z07:00")
			nodeStatus.StartedAt = &startedAt
		}

		if node.FinishedAt != nil {
			finishedAt := node.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
			nodeStatus.FinishedAt = &finishedAt
		}

		batchMap[node.Batch] = append(batchMap[node.Batch], nodeStatus)
	}

	// 构建batches数组
	var batches []BatchNodeStatus
	for batch, nodes := range batchMap {
		batches = append(batches, BatchNodeStatus{
			Batch: batch,
			Nodes: nodes,
		})
	}

	// 按batch排序
	for i := 0; i < len(batches); i++ {
		for j := i + 1; j < len(batches); j++ {
			if batches[i].Batch > batches[j].Batch {
				batches[i], batches[j] = batches[j], batches[i]
			}
		}
	}

	return &GetReleaseResponse{
		ReleaseID:    task.ID,
		Version:      task.Version,
		Status:       string(task.Status),
		TotalNodes:   task.TotalNodes,
		SuccessNodes: task.SuccessNodes,
		FailedNodes:  task.FailedNodes,
		Batches:      batches,
		CreatedAt:    task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    task.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
