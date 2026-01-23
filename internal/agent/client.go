package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an HTTP client for communicating with agents
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a new agent client
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		token: token,
	}
}

// ExecuteTaskRequest represents a task execution request
type ExecuteTaskRequest struct {
	RequestID string      `json:"requestId"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
}

// ExecuteTaskResponse represents a task execution response
type ExecuteTaskResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RequestID string `json:"requestId"`
		Status    string `json:"status"`
		Message   string `json:"message"`
	} `json:"data"`
}

// ExecuteTask sends a task execution request to an agent
func (c *Client) ExecuteTask(agentURL string, req ExecuteTaskRequest) (*ExecuteTaskResponse, error) {
	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", agentURL+"/agent/v1/tasks/execute", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status code
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var resp ExecuteTaskResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response code
	if resp.Code != 0 {
		return nil, fmt.Errorf("agent returned error code %d: %s", resp.Code, resp.Message)
	}

	return &resp, nil
}
