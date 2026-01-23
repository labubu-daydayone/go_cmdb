package release

import (
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// SelectOnlineNodes 选择在线节点
// 规则：enabled=1, status='online', 按id升序
func SelectOnlineNodes(db *gorm.DB) ([]model.Node, error) {
	var nodes []model.Node
	err := db.Where("enabled = ? AND status = ?", 1, "online").
		Order("id ASC").
		Find(&nodes).Error
	return nodes, err
}

// AllocateBatches 分配批次
// 规则：batch=1取第1个节点，batch=2取剩余节点
// 若只有1个节点，仅batch=1，不生成batch=2
func AllocateBatches(nodes []model.Node) []BatchAllocation {
	if len(nodes) == 0 {
		return nil
	}

	batches := []BatchAllocation{
		{
			Batch:   1,
			NodeIDs: []int{nodes[0].ID},
		},
	}

	if len(nodes) > 1 {
		remainingIDs := make([]int, len(nodes)-1)
		for i := 1; i < len(nodes); i++ {
			remainingIDs[i-1] = nodes[i].ID
		}
		batches = append(batches, BatchAllocation{
			Batch:   2,
			NodeIDs: remainingIDs,
		})
	}

	return batches
}
