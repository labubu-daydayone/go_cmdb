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
