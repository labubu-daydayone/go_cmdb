package upstream

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// NodeSelector 节点选择器
type NodeSelector struct {
	db *gorm.DB
}

// NewNodeSelector 创建节点选择器
func NewNodeSelector(db *gorm.DB) *NodeSelector {
	return &NodeSelector{db: db}
}

// SelectNodesForWebsite 为 website 选择目标节点
func (s *NodeSelector) SelectNodesForWebsite(websiteID int64) ([]int, error) {
	// 1. 查询 website
	var website model.Website
	if err := s.db.First(&website, websiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.ErrNotFound("website not found")
		}
		return nil, httpx.ErrDatabaseError("failed to query website", err)
	}

	// 2. 按 line_group 选择（website 只有 lineGroupId）
	if website.LineGroupID > 0 {
		return s.selectNodesByLineGroup(int64(website.LineGroupID))
	}

	// 4. 否则选择所有在线节点
	return s.selectAllOnlineNodes()
}

// selectNodesByLineGroup 按 line_group 选择节点
func (s *NodeSelector) selectNodesByLineGroup(lineGroupID int64) ([]int, error) {
	// line_group 关联的节点通过 node_ips 的 line_group_id 字段
	var nodeIDs []int
	err := s.db.Model(&model.NodeIP{}).
		Select("DISTINCT node_id").
		Where("line_group_id = ?", lineGroupID).
		Scan(&nodeIDs).Error
	if err != nil {
		return nil, httpx.ErrDatabaseError("failed to query line group nodes", err)
	}

	// 过滤：只保留 enabled=true 且 agent_status=online 的节点
	return s.filterOnlineNodes(nodeIDs)
}

// selectAllOnlineNodes 选择所有在线节点
func (s *NodeSelector) selectAllOnlineNodes() ([]int, error) {
	var nodeIDs []int
	err := s.db.Model(&model.Node{}).
		Select("id").
		Where("enabled = ? AND agent_status = ?", true, "online").
		Scan(&nodeIDs).Error
	if err != nil {
		return nil, httpx.ErrDatabaseError("failed to query online nodes", err)
	}
	return nodeIDs, nil
}

// filterOnlineNodes 过滤出在线且启用的节点
func (s *NodeSelector) filterOnlineNodes(nodeIDs []int) ([]int, error) {
	if len(nodeIDs) == 0 {
		return []int{}, nil
	}

	var filteredIDs []int
	err := s.db.Model(&model.Node{}).
		Select("id").
		Where("id IN ? AND enabled = ? AND agent_status = ?", nodeIDs, true, "online").
		Scan(&filteredIDs).Error
	if err != nil {
		return nil, httpx.ErrDatabaseError("failed to filter online nodes", err)
	}
	return filteredIDs, nil
}
