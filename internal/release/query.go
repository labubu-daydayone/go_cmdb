package release

import (
	"database/sql"
	"time"

	"go_cmdb/internal/model"
)

// ListReleasesRequest 列表查询请求
type ListReleasesRequest struct {
	Status   string `form:"status"`   // pending/running/success/failed/paused
	Page     int    `form:"page"`     // 页码，默认1
	PageSize int    `form:"pageSize"` // 每页数量，默认20，最大100
}

// ReleaseListItem 发布任务列表项
type ReleaseListItem struct {
	ID            int64     `json:"id"`
	Type          string    `json:"type"`
	Target        string    `json:"target"`
	Version       int64     `json:"version"`
	Status        string    `json:"status"`
	TotalNodes    int       `json:"totalNodes"`
	SuccessNodes  int       `json:"successNodes"`
	FailedNodes   int       `json:"failedNodes"`
	SkippedNodes  int       `json:"skippedNodes"`
	CurrentBatch  int       `json:"currentBatch"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ListReleasesResponse 列表查询响应
type ListReleasesResponse struct {
	Items    []ReleaseListItem `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// ListReleases 查询发布任务列表
func (s *Service) ListReleases(req *ListReleasesRequest) (*ListReleasesResponse, error) {
	// 参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 构建查询
	query := s.db.Model(&model.ReleaseTask{})
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 查询列表
	var tasks []model.ReleaseTask
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Limit(req.PageSize).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, err
	}

	// 构建响应
	items := make([]ReleaseListItem, 0, len(tasks))
	for _, task := range tasks {
		// 计算skippedNodes
		var skippedNodes int64
		s.db.Model(&model.ReleaseTaskNode{}).
			Where("release_task_id = ? AND status = ?", task.ID, model.ReleaseTaskNodeStatusSkipped).
			Count(&skippedNodes)

		// 计算currentBatch
		var currentBatch sql.NullInt64
		s.db.Model(&model.ReleaseTaskNode{}).
			Where("release_task_id = ? AND status IN ?", task.ID, []model.ReleaseTaskNodeStatus{
				model.ReleaseTaskNodeStatusPending,
				model.ReleaseTaskNodeStatusRunning,
				model.ReleaseTaskNodeStatusFailed,
			}).
			Select("MIN(batch)").
			Scan(&currentBatch)

		currentBatchValue := 0
		if currentBatch.Valid {
			currentBatchValue = int(currentBatch.Int64)
		}

		items = append(items, ReleaseListItem{
			ID:            task.ID,
			Type:          string(task.Type),
			Target:        string(task.Target),
			Version:       task.Version,
			Status:        string(task.Status),
			TotalNodes:    task.TotalNodes,
			SuccessNodes:  task.SuccessNodes,
			FailedNodes:   task.FailedNodes,
			SkippedNodes:  int(skippedNodes),
			CurrentBatch:  currentBatchValue,
			CreatedAt:     task.CreatedAt,
			UpdatedAt:     task.UpdatedAt,
		})
	}

	return &ListReleasesResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ReleaseDetailNode 发布任务节点详情
type ReleaseDetailNode struct {
	NodeID     int        `json:"nodeId"`
	NodeName   string     `json:"nodeName"`
	Status     string     `json:"status"`
	ErrorMsg   string     `json:"errorMsg"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

// ReleaseDetailBatch 发布任务批次详情
type ReleaseDetailBatch struct {
	Batch int                 `json:"batch"`
	Nodes []ReleaseDetailNode `json:"nodes"`
}

// ReleaseDetail 发布任务详情
type ReleaseDetail struct {
	ID            int64     `json:"id"`
	Type          string    `json:"type"`
	Target        string    `json:"target"`
	Version       int64     `json:"version"`
	Status        string    `json:"status"`
	TotalNodes    int       `json:"totalNodes"`
	SuccessNodes  int       `json:"successNodes"`
	FailedNodes   int       `json:"failedNodes"`
	SkippedNodes  int       `json:"skippedNodes"`
	CurrentBatch  int       `json:"currentBatch"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// GetReleaseDetailResponse 详情查询响应
type GetReleaseDetailResponse struct {
	Release ReleaseDetail        `json:"release"`
	Batches []ReleaseDetailBatch `json:"batches"`
}

// NodeWithTask 节点与任务关联信息
type NodeWithTask struct {
	NodeID     int
	NodeName   string
	Batch      int
	Status     string
	ErrorMsg   sql.NullString
	StartedAt  sql.NullTime
	FinishedAt sql.NullTime
}

// GetReleaseDetail 查询发布任务详情
func (s *Service) GetReleaseDetail(id int64) (*GetReleaseDetailResponse, error) {
	// 查询release_task
	var task model.ReleaseTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}

	// 计算skippedNodes
	var skippedNodes int64
	s.db.Model(&model.ReleaseTaskNode{}).
		Where("release_task_id = ? AND status = ?", task.ID, model.ReleaseTaskNodeStatusSkipped).
		Count(&skippedNodes)

	// 计算currentBatch
	var currentBatch sql.NullInt64
	s.db.Model(&model.ReleaseTaskNode{}).
		Where("release_task_id = ? AND status IN ?", task.ID, []model.ReleaseTaskNodeStatus{
			model.ReleaseTaskNodeStatusPending,
			model.ReleaseTaskNodeStatusRunning,
			model.ReleaseTaskNodeStatusFailed,
		}).
		Select("MIN(batch)").
		Scan(&currentBatch)

	currentBatchValue := 0
	if currentBatch.Valid {
		currentBatchValue = int(currentBatch.Int64)
	}

	// 查询release_task_nodes（JOIN nodes获取nodeName）
	var nodesWithTask []NodeWithTask
	err := s.db.Table("release_task_nodes rtn").
		Select("rtn.node_id, n.name as node_name, rtn.batch, rtn.status, rtn.error_msg, rtn.started_at, rtn.finished_at").
		Joins("LEFT JOIN nodes n ON rtn.node_id = n.id").
		Where("rtn.release_task_id = ?", id).
		Order("rtn.batch ASC, rtn.node_id ASC").
		Scan(&nodesWithTask).Error
	if err != nil {
		return nil, err
	}

	// 按batch分组
	batchMap := make(map[int][]ReleaseDetailNode)
	for _, nwt := range nodesWithTask {
		errorMsg := ""
		if nwt.ErrorMsg.Valid {
			errorMsg = nwt.ErrorMsg.String
		}

		var startedAt *time.Time
		if nwt.StartedAt.Valid {
			startedAt = &nwt.StartedAt.Time
		}

		var finishedAt *time.Time
		if nwt.FinishedAt.Valid {
			finishedAt = &nwt.FinishedAt.Time
		}

		node := ReleaseDetailNode{
			NodeID:     nwt.NodeID,
			NodeName:   nwt.NodeName,
			Status:     nwt.Status,
			ErrorMsg:   errorMsg,
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
		}

		batchMap[nwt.Batch] = append(batchMap[nwt.Batch], node)
	}

	// 构建batches（按batch升序）
	batches := make([]ReleaseDetailBatch, 0, len(batchMap))
	for batch := 1; batch <= len(batchMap); batch++ {
		if nodes, ok := batchMap[batch]; ok {
			batches = append(batches, ReleaseDetailBatch{
				Batch: batch,
				Nodes: nodes,
			})
		}
	}

	return &GetReleaseDetailResponse{
		Release: ReleaseDetail{
			ID:            task.ID,
			Type:          string(task.Type),
			Target:        string(task.Target),
			Version:       task.Version,
			Status:        string(task.Status),
			TotalNodes:    task.TotalNodes,
			SuccessNodes:  task.SuccessNodes,
			FailedNodes:   task.FailedNodes,
			SkippedNodes:  int(skippedNodes),
			CurrentBatch:  currentBatchValue,
			CreatedAt:     task.CreatedAt,
			UpdatedAt:     task.UpdatedAt,
		},
		Batches: batches,
	}, nil
}
