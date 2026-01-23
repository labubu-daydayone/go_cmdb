package agentclient

import "encoding/json"

// DispatchRequest 派发任务请求
type DispatchRequest struct {
	TaskID  string          `json:"taskId"`
	Type    string          `json:"type"`
	Version int64           `json:"version,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// DispatchResponse 派发任务响应
type DispatchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID string `json:"taskId"`
		Status string `json:"status"`
	} `json:"data"`
}

// QueryResponse 查询任务响应
type QueryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID     string  `json:"taskId"`
		Type       string  `json:"type"`
		Status     string  `json:"status"`
		LastError  string  `json:"lastError,omitempty"`
		CreatedAt  string  `json:"createdAt"`
		StartedAt  *string `json:"startedAt,omitempty"`
		FinishedAt *string `json:"finishedAt,omitempty"`
	} `json:"data"`
}
