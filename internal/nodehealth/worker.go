package nodehealth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"go_cmdb/internal/model"
)

// Worker for node health checks	ype Worker struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	db                   *gorm.DB
	client               *http.Client
	logger               *logrus.Entry
	interval             time.Duration
	offlineFailThreshold int
	concurrency          int
}

// Config holds the configuration for the health check worker	ype Config struct {
	DB                   *gorm.DB
	Client               *http.Client
	Logger               *logrus.Entry
	IntervalSec          int
	OfflineFailThreshold int
	Concurrency          int
}

// PingResponse is the expected response from the agent's ping endpoint	ype PingResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CheckResult holds the result of a single manual health check	ype CheckResult struct {
	NodeID     int        `json:"nodeId"`
	OK         bool       `json:"ok"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
	Error      string     `json:"error"`
}

// NewWorker creates a new health check worker
func NewWorker(cfg *Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		ctx:                  ctx,
		cancel:               cancel,
		db:                   cfg.DB,
		client:               cfg.Client,
		logger:               cfg.Logger.WithField("component", "node-health-worker"),
		interval:             time.Duration(cfg.IntervalSec) * time.Second,
		offlineFailThreshold: cfg.OfflineFailThreshold,
		concurrency:          cfg.Concurrency,
	}
}

// Start begins the periodic health checks
func (w *Worker) Start() {
	w.logger.Info("Starting node health worker...")
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.runHealthChecks()
			case <-w.ctx.Done():
				w.logger.Info("Stopping node health worker...")
				return
			}
		}
	}()
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.cancel()
}

func (w *Worker) runHealthChecks() {
	var nodes []model.Node
	if err := w.db.Where("enabled = ?", true).Find(&nodes).Error; err != nil {
		w.logger.Errorf("Failed to fetch nodes for health check: %v", err)
		return
	}

	if len(nodes) == 0 {
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, w.concurrency)

	for _, node := range nodes {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(n model.Node) {
			defer wg.Done()
			defer func() { <-semaphore }()
			w.checkNode(&n)
		}(node)
	}

	wg.Wait()
}

func (w *Worker) checkNode(node *model.Node) {
	url := fmt.Sprintf("https://%s:%d/api/v1/ping", node.MainIP, node.AgentPort)
	req, err := http.NewRequestWithContext(w.ctx, "GET", url, nil)
	if err != nil {
		w.handleFailure(node, fmt.Errorf("failed to create request: %w", err))
		return
	}

	resp, err := w.client.Do(req)
	if err != nil {
		w.handleFailure(node, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.handleFailure(node, fmt.Errorf("unexpected status code: %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.handleFailure(node, fmt.Errorf("failed to read response body: %w", err))
		return
	}

	var pingResp PingResponse
	if err := json.Unmarshal(body, &pingResp); err != nil {
		w.handleFailure(node, fmt.Errorf("failed to unmarshal json: %w", err))
		return
	}

	if pingResp.Code != 0 {
		w.handleFailure(node, fmt.Errorf("agent returned non-zero code: %d, message: %s", pingResp.Code, pingResp.Message))
		return
	}

	w.handleSuccess(node)
}

func (w *Worker) handleSuccess(node *model.Node) {
	updates := map[string]interface{}{
		"last_seen_at":       time.Now(),
		"last_health_error":  nil,
		"health_fail_count":  0,
	}

	if node.Status != model.NodeStatusMaintenance {
		updates["status"] = model.NodeStatusOnline
	}

	if err := w.db.Model(node).Updates(updates).Error; err != nil {
		w.logger.Errorf("Failed to update node %d on success: %v", node.ID, err)
	}
}

func (w *Worker) handleFailure(node *model.Node, err error) {
	errorMsg := err.Error()
	if len(errorMsg) > 255 {
		errorMsg = errorMsg[:255]
	}

	newFailCount := node.HealthFailCount + 1
	updates := map[string]interface{}{
		"last_health_error": &errorMsg,
		"health_fail_count": newFailCount,
	}

	if node.Status != model.NodeStatusMaintenance && newFailCount >= w.offlineFailThreshold {
		updates["status"] = model.NodeStatusOffline
	}

	if err := w.db.Model(node).Updates(updates).Error; err != nil {
		w.logger.Errorf("Failed to update node %d on failure: %v", node.ID, err)
	}
}

// CheckNodes performs an immediate health check on a list of nodes
func (w *Worker) CheckNodes(nodeIDs []int) []CheckResult {
	var nodes []model.Node
	if err := w.db.Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		w.logger.Errorf("Failed to fetch nodes for manual check: %v", err)
		return nil
	}

	results := make([]CheckResult, len(nodes))
	var wg sync.WaitGroup
	resultChan := make(chan CheckResult, len(nodes))

	for _, node := range nodes {
		wg.Add(1)
		go func(n model.Node) {
			defer wg.Done()
			w.checkNode(&n)

			// Re-fetch node to get updated status
			var updatedNode model.Node
			w.db.First(&updatedNode, n.ID)

			result := CheckResult{
				NodeID:     int(updatedNode.ID),
				Status:     string(updatedNode.Status),
				LastSeenAt: updatedNode.LastSeenAt,
			}
			if updatedNode.LastHealthError != nil {
				result.Error = *updatedNode.LastHealthError
			} else {
				result.Error = ""
			}
			result.OK = result.Error == ""

			resultChan <- result
		}(node)
	}

	wg.Wait()
	close(resultChan)

	i := 0
	for res := range resultChan {
		results[i] = res
		i++
	}

	return results
}
