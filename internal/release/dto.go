package release

// CreateReleaseRequest 创建发布任务请求
type CreateReleaseRequest struct {
	Target string  `json:"target" binding:"required"` // 目标类型（cdn）
	Reason *string `json:"reason"`                    // 原因（可选）
}

// BatchAllocation 批次分配
type BatchAllocation struct {
	Batch   int   `json:"batch"`   // 批次号
	NodeIDs []int `json:"nodeIds"` // 节点ID列表
}

// CreateReleaseResponse 创建发布任务响应
type CreateReleaseResponse struct {
	ReleaseID  int64              `json:"releaseId"`  // 发布任务ID
	Version    int64              `json:"version"`    // 版本号
	TotalNodes int                `json:"totalNodes"` // 总节点数
	Batches    []BatchAllocation  `json:"batches"`    // 批次分配
}
