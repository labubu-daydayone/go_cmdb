package agentclient

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

// Client Agent客户端
type Client struct {
	httpClient *http.Client
	config     *config.Config
}

// NewClient 创建Agent客户端（支持mTLS）
func NewClient(cfg *config.Config) (*Client, error) {
	if !cfg.MTLS.Enabled {
		return nil, fmt.Errorf("mTLS is required but not enabled")
	}

	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(cfg.MTLS.ClientCert, cfg.MTLS.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// 加载CA证书
	caCert, err := os.ReadFile(cfg.MTLS.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// 创建TLS配置
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false, // 必须验证服务端证书
		MinVersion:         tls.VersionTLS12,
	}

	// 创建HTTP客户端（支持mTLS + 超时）
	httpClient := &http.Client{
		Timeout: 60 * time.Second, // 请求超时60s
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: 15 * time.Second, // 连接超时15s
		},
	}

	return &Client{
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// Dispatch 派发apply_config任务到Agent
// nodeIP: Agent的IP地址
// version: 配置版本号
// 返回: taskID（用于后续查询）和错误
func (c *Client) Dispatch(nodeIP string, version int64) (string, error) {
	// 构造Agent URL（假设Agent监听8443端口）
	agentURL := fmt.Sprintf("https://%s:8443", nodeIP)

	// 生成taskID（使用nodeIP + version）
	taskID := fmt.Sprintf("apply_config_%s_%d", nodeIP, version)

	// 构造请求
	req := DispatchRequest{
		TaskID:  taskID,
		Type:    "apply_config",
		Version: version,
		Payload: json.RawMessage("{}"), // apply_config不需要额外payload
	}

	// 序列化请求
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", agentURL+"/tasks/dispatch", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request to agent %s (mTLS handshake may have failed): %w", nodeIP, err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent %s returned status %d: %s", nodeIP, httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp DispatchResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查响应code
	if resp.Code != 0 {
		return "", fmt.Errorf("agent %s returned error code %d: %s", nodeIP, resp.Code, resp.Message)
	}

	return resp.Data.TaskID, nil
}

// Query 查询任务状态
// nodeIP: Agent的IP地址
// taskID: 任务ID
// 返回: status（pending/running/success/failed）、lastError和错误
func (c *Client) Query(nodeIP string, taskID string) (string, string, error) {
	// 构造Agent URL
	agentURL := fmt.Sprintf("https://%s:8443", nodeIP)

	// 创建HTTP请求
	httpReq, err := http.NewRequest("GET", agentURL+"/tasks/"+taskID, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request to agent %s: %w", nodeIP, err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("task %s not found on agent %s", taskID, nodeIP)
	}

	if httpResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("agent %s returned status %d: %s", nodeIP, httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp QueryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查响应code
	if resp.Code != 0 {
		return "", "", fmt.Errorf("agent %s returned error code %d: %s", nodeIP, resp.Code, resp.Message)
	}

	return resp.Data.Status, resp.Data.LastError, nil
}
