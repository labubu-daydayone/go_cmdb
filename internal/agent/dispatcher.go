package agent

import (
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/config"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Dispatcher handles task dispatching to agents
type Dispatcher struct {
	db     *gorm.DB
	client *Client
}

// NewDispatcher creates a new task dispatcher with mTLS support
func NewDispatcher(db *gorm.DB, cfg *config.Config) (*Dispatcher, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent client: %w", err)
	}

	return &Dispatcher{
		db:     db,
		client: client,
	}, nil
}

// DispatchTask dispatches a task to an agent with identity verification
func (d *Dispatcher) DispatchTask(task *model.AgentTask) error {
	// Get node information
	var node model.Node
	if err := d.db.First(&node, task.NodeID).Error; err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Verify agent identity (MUST check before dispatching)
	var identity model.AgentIdentity
	if err := d.db.Where("node_id = ?", node.ID).First(&identity).Error; err != nil {
		// Identity not found
		task.Status = model.TaskStatusFailed
		task.LastError = "agent identity not found"
		task.Attempts++
		if saveErr := d.db.Save(task).Error; saveErr != nil {
			log.Printf("Failed to update task status: %v", saveErr)
		}
		// Update config_versions status if task type is apply_config
		if task.Type == model.TaskTypeApplyConfig {
			d.updateConfigVersionStatus(task, model.ConfigVersionStatusFailed, task.LastError)
		}
		return fmt.Errorf("agent identity not found for node %d", node.ID)
	}

	// Check identity status
	if identity.Status != model.AgentIdentityStatusActive {
		// Identity revoked
		task.Status = model.TaskStatusFailed
		task.LastError = fmt.Sprintf("agent identity is %s", identity.Status)
		task.Attempts++
		if saveErr := d.db.Save(task).Error; saveErr != nil {
			log.Printf("Failed to update task status: %v", saveErr)
		}
		// Update config_versions status if task type is apply_config
		if task.Type == model.TaskTypeApplyConfig {
			d.updateConfigVersionStatus(task, model.ConfigVersionStatusFailed, task.LastError)
		}
		return fmt.Errorf("agent identity is %s for node %d", identity.Status, node.ID)
	}

	// Build agent URL (HTTPS only)
	agentURL := fmt.Sprintf("https://%s:%d", node.MainIP, node.AgentPort)

	// Update task status to running
	task.Status = model.TaskStatusRunning
	if err := d.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Parse payload
	var payload interface{}
	if task.Payload != "" {
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse payload: %w", err)
		}
	}

	// Prepare request
	req := ExecuteTaskRequest{
		RequestID: task.RequestID,
		Type:      task.Type,
		Payload:   payload,
	}

	// Send request to agent via mTLS
	resp, err := d.client.ExecuteTask(agentURL, req)
	if err != nil {
		// Task failed (could be mTLS handshake failure or execution error)
		task.Status = model.TaskStatusFailed
		task.LastError = err.Error()
		task.Attempts++
		if saveErr := d.db.Save(task).Error; saveErr != nil {
			log.Printf("Failed to update task status after error: %v", saveErr)
		}
		// Update config_versions status if task type is apply_config
		if task.Type == model.TaskTypeApplyConfig {
			d.updateConfigVersionStatus(task, model.ConfigVersionStatusFailed, task.LastError)
		}
		return err
	}

	// Task succeeded
	task.Status = model.TaskStatusSuccess
	task.LastError = ""
	task.Attempts++
	if err := d.db.Save(task).Error; err != nil {
		log.Printf("Failed to update task status after success: %v", err)
		return err
	}

	// Update config_versions status if task type is apply_config
	if task.Type == model.TaskTypeApplyConfig {
		d.updateConfigVersionStatus(task, model.ConfigVersionStatusApplied, "")
	}

	log.Printf("Task %d dispatched successfully: %s", task.ID, resp.Data.Message)
	return nil
}

// updateConfigVersionStatus updates config_versions status based on task result
func (d *Dispatcher) updateConfigVersionStatus(task *model.AgentTask, status string, lastError string) {
	// Parse payload to extract version
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		log.Printf("Failed to parse task payload for config version update: %v", err)
		return
	}

	// Extract version from payload
	versionFloat, ok := payload["version"].(float64)
	if !ok {
		log.Printf("Failed to extract version from task payload")
		return
	}
	version := int64(versionFloat)

	// Update config_versions status
	updates := map[string]interface{}{
		"status": status,
	}

	if status == model.ConfigVersionStatusFailed && lastError != "" {
		// Truncate last_error to 255 characters (database field limit)
		if len(lastError) > 255 {
			lastError = lastError[:252] + "..."
		}
		updates["last_error"] = lastError
	}

	if err := d.db.Model(&model.ConfigVersion{}).
		Where("version = ?", version).
		Updates(updates).Error; err != nil {
		log.Printf("Failed to update config_versions status: %v", err)
	}
}
