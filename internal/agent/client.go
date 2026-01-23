package agent

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go_cmdb/internal/config"
)

// Client represents an HTTP client for communicating with agents
type Client struct {
	httpClient *http.Client
	config     *config.Config
}

// NewClient creates a new agent client with mTLS support
func NewClient(cfg *config.Config) (*Client, error) {
	if !cfg.MTLS.Enabled {
		return nil, fmt.Errorf("mTLS is required but not enabled")
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(cfg.MTLS.ClientCert, cfg.MTLS.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(cfg.MTLS.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false, // MUST verify server certificate
		MinVersion:         tls.VersionTLS12,
	}

	// Create HTTP client with mTLS
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		httpClient: httpClient,
		config:     cfg,
	}, nil
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

// ExecuteTask sends a task execution request to an agent via mTLS
func (c *Client) ExecuteTask(agentURL string, req ExecuteTaskRequest) (*ExecuteTaskResponse, error) {
	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request (HTTPS only)
	httpReq, err := http.NewRequest("POST", agentURL+"/agent/v1/tasks/execute", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (no Bearer token, mTLS only)
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request (mTLS handshake may have failed): %w", err)
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
