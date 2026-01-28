package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client HTTP 客户端，用于与控制面通信
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient 创建客户端
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UpdateHeartbeat 更新 Worker 心跳
func (c *Client) UpdateHeartbeat(ctx context.Context, workerName string) error {
	url := fmt.Sprintf("%s/api/v1/workers/%s/heartbeat", c.BaseURL, workerName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetWorkerConfig 获取 Worker 配置
func (c *Client) GetWorkerConfig(ctx context.Context, workerName string) (*WorkerConfigResponse, error) {
	url := fmt.Sprintf("%s/api/v1/workers/%s", c.BaseURL, workerName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Item WorkerConfigResponse `json:"item"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result.Item, nil
}

// WorkerConfigResponse Worker 配置响应
type WorkerConfigResponse struct {
	WorkerName        string         `json:"worker_name"`
	Concurrency       int            `json:"concurrency"`
	Queues            map[string]int `json:"queues"`
	DefaultRetryCount int            `json:"default_retry_count"`
	DefaultTimeout    int            `json:"default_timeout"`
	IsEnabled         bool           `json:"is_enabled"`
}

// EnqueueTask 直接向控制面提交任务（bypass Asynq）
func (c *Client) EnqueueTask(ctx context.Context, req EnqueueTaskRequest) (*EnqueueTaskResponse, error) {
	url := fmt.Sprintf("%s/api/v1/tasks", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result EnqueueTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// EnqueueTaskRequest 任务入队请求
type EnqueueTaskRequest struct {
	WorkerName   string          `json:"worker_name"`
	Queue        string          `json:"queue"`
	TaskID       string          `json:"task_id,omitempty"`
	Payload      json.RawMessage `json:"payload"`
	DelaySeconds int             `json:"delay_seconds,omitempty"`
}

// EnqueueTaskResponse 任务入队响应
type EnqueueTaskResponse struct {
	TaskID      string `json:"task_id"`
	WorkerName  string `json:"worker_name"`
	Queue       string `json:"queue"`
	AsynqTaskID string `json:"asynq_task_id"`
	Status      string `json:"status"`
}
