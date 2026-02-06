package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"go_cmdb/internal/db"
	"go_cmdb/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
)

// AgentTaskPayload defines the structure of the JSON payload in an agent task.
type AgentTaskPayload struct {
	ReleaseTaskID uint `json:"releaseTaskId"`
}

// UpdateTaskStatus handles the entire logic of updating an agent task status
// and propagating the result to the parent release task.
func UpdateTaskStatus(nodeID, taskID uint, apiStatus, errorMessage string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Find the agent task and validate its ownership and current state.
		var agentTask model.AgentTask
		if err := tx.Where("id = ? AND node_id = ?", taskID, nodeID).First(&agentTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("task not found or not assigned to this node")
			}
			return err
		}

		if agentTask.Status != "running" {
			return fmt.Errorf("task is not in a running state, current status: %s", agentTask.Status)
		}

		// 2. Update the agent_task status.
		dbStatus := apiStatus
		if (apiStatus == "succeeded") {
		    dbStatus = "success"
		}

		updateData := map[string]interface{}{
			"status":     dbStatus,
			"updated_at": gorm.Expr("NOW()"),
		}
		if dbStatus == "failed" {
			updateData["last_error"] = errorMessage
		}

		if err := tx.Model(&agentTask).Updates(updateData).Error; err != nil {
			return err
		}

		// 3. Propagate the result to the release_task.
		var payload AgentTaskPayload
		if err := json.Unmarshal([]byte(agentTask.Payload), &payload); err != nil {
			log.Printf("[Error] Failed to unmarshal agent task payload for task %d: %v", agentTask.ID, err)
			return fmt.Errorf("invalid task payload")
		}

		if payload.ReleaseTaskID == 0 {
			log.Printf("[Info] No releaseTaskId in payload for agent task %d. Skipping release task update.", agentTask.ID)
			return nil // Not every agent task belongs to a release task.
		}

		// Lock the release_task row for atomic update.
		var releaseTask model.ReleaseTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&releaseTask, payload.ReleaseTaskID).Error; err != nil {
			return fmt.Errorf("failed to find and lock release task %d: %w", payload.ReleaseTaskID, err)
		}

			// Update success/failed node counts.
			updates := make(map[string]interface{})
			if dbStatus == "success" {
				updates["success_nodes"] = gorm.Expr("success_nodes + 1")
			} else {
				updates["failed_nodes"] = gorm.Expr("failed_nodes + 1")
				errorMsg := fmt.Sprintf("Node %d failed: %s", nodeID, errorMessage)
				updates["last_error"] = errorMsg
			}

			// Re-fetch the updated counts to determine final status.
			if err := tx.Model(&model.ReleaseTask{}).Where("id = ?", payload.ReleaseTaskID).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update release task counts: %w", err)
			}

			// Re-fetch to get the updated counts.
			if err := tx.First(&releaseTask, payload.ReleaseTaskID).Error; err != nil {
				return fmt.Errorf("failed to re-fetch release task: %w", err)
			}

			// 4. Check if the release task is complete.
			if (releaseTask.SuccessNodes + releaseTask.FailedNodes) >= releaseTask.TotalNodes {
				var finalStatus model.ReleaseTaskStatus
				if releaseTask.FailedNodes > 0 {
					finalStatus = model.ReleaseTaskStatusFailed
				} else {
					finalStatus = model.ReleaseTaskStatusSuccess
				}
				return tx.Model(&model.ReleaseTask{}).Where("id = ?", payload.ReleaseTaskID).Update("status", finalStatus).Error
			}

			return nil
	})
}
