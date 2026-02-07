package service

import (
	"fmt"
	"log"

	"go_cmdb/internal/model"
)

// DispatchPendingResult 补派发结果
type DispatchPendingResult struct {
	PendingReleaseTaskTouchedCount int
	DispatchedCreatedCount         int
	DispatchedSkippedCount         int
	DispatchedBeforePull           bool
}

// EnsureDispatchPendingForNode 为指定节点补派发离线期间的 pending release_tasks
func (d *AgentTaskDispatcher) EnsureDispatchPendingForNode(nodeID int64) (*DispatchPendingResult, error) {
	result := &DispatchPendingResult{
		DispatchedBeforePull: true,
	}

	// 1. 查询节点所属的 line_groups
	var lineGroupIDs []int
	err := d.db.Table("line_groups").
		Select("DISTINCT line_groups.id").
		Joins("JOIN node_group_ips ON line_groups.node_group_id = node_group_ips.node_group_id").
		Joins("JOIN node_ips ON node_group_ips.ip_id = node_ips.id").
		Where("node_ips.node_id = ?", nodeID).
		Where("node_ips.enabled = ?", true).
		Where("node_ips.status = ?", "active").
		Pluck("line_groups.id", &lineGroupIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query line_groups for node %d: %w", nodeID, err)
	}

	if len(lineGroupIDs) == 0 {
		log.Printf("[DispatchPending] Node %d has no associated line_groups", nodeID)
		return result, nil
	}

	log.Printf("[DispatchPending] Node %d belongs to line_groups: %v", nodeID, lineGroupIDs)

	// 2. 查询这些 line_groups 对应的 pending/running release_tasks（target_type=website）
	var releaseTasks []model.ReleaseTask
	err = d.db.Table("release_tasks").
		Select("release_tasks.*").
		Joins("JOIN websites ON release_tasks.target_id = websites.id AND release_tasks.target_type = 'website'").
		Where("websites.line_group_id IN ?", lineGroupIDs).
		Where("release_tasks.status IN ?", []string{"pending", "running"}).
		Order("release_tasks.id ASC").
		Find(&releaseTasks).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query pending release_tasks: %w", err)
	}

	if len(releaseTasks) == 0 {
		log.Printf("[DispatchPending] No pending release_tasks found for node %d", nodeID)
		return result, nil
	}

	log.Printf("[DispatchPending] Found %d pending release_tasks for node %d", len(releaseTasks), nodeID)

	// 3. 对每个 release_task 执行补派发
	for _, releaseTask := range releaseTasks {
		result.PendingReleaseTaskTouchedCount++

		// 调用 EnsureDispatched 进行补派发
		dispatchResult, err := d.EnsureDispatched(int64(releaseTask.ID), releaseTask.TargetID, fmt.Sprintf("resume-sync-node-%d", nodeID))
		if err != nil {
			log.Printf("[DispatchPending] Failed to dispatch release_task %d: %v", releaseTask.ID, err)
			continue
		}

		// 统计补派发结果
		result.DispatchedCreatedCount += dispatchResult.Created
		result.DispatchedSkippedCount += dispatchResult.Skipped

		log.Printf("[DispatchPending] Dispatched release_task %d for node %d: created=%d, skipped=%d",
			releaseTask.ID, nodeID, dispatchResult.Created, dispatchResult.Skipped)
	}

	log.Printf("[DispatchPending] Completed for node %d: touched=%d, created=%d, skipped=%d",
		nodeID, result.PendingReleaseTaskTouchedCount, result.DispatchedCreatedCount, result.DispatchedSkippedCount)

	return result, nil
}
