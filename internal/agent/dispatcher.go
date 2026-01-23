package agent

import (
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Dispatcher handles task dispatching to agents
type Dispatcher struct {
	db     *gorm.DB
	client *Client
}

// NewDispatcher creates a new task dispatcher
func NewDispatcher(db *gorm.DB, agentToken string) *Dispatcher {
	return &Dispatcher{
		db:     db,
		client: NewClient(agentToken),
	}
}

// DispatchTask dispatches a task to an agent
func (d *Dispatcher) DispatchTask(task *model.AgentTask) error {
	// Get node information
	var node model.Node
	if err := d.db.First(&node, task.NodeID).Error; err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Build agent URL
	agentURL := fmt.Sprintf("http://%s:%d", node.MainIP, node.AgentPort)

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

	// Send request to agent
	resp, err := d.client.ExecuteTask(agentURL, req)
	if err != nil {
		// Task failed
		task.Status = model.TaskStatusFailed
		task.LastError = err.Error()
		task.Attempts++
		if err := d.db.Save(task).Error; err != nil {
			log.Printf("Failed to update task status after error: %v", err)
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

	log.Printf("Task %d dispatched successfully: %s", task.ID, resp.Data.Message)
	return nil
}
