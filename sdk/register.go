package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WorkerConfig Worker 注册配置
type WorkerConfig struct {
	WorkerName        string             `json:"worker_name"`
	BaseURL           string             `json:"base_url"`
	RedisAddr         string             `json:"redis_addr"`
	QueueGroups       []QueueGroupConfig `json:"queue_groups"` // 队列组配置
	DefaultRetryCount int                `json:"default_retry_count"`
	DefaultTimeout    int                `json:"default_timeout"` // seconds
	DefaultDelay      int                `json:"default_delay"`   // seconds
}

type Registrar struct {
	ControlPlaneURL string
	HTTPClient      *http.Client
}

func (r Registrar) client() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Second}
}

func (r Registrar) enabled() bool {
	return r.ControlPlaneURL != ""
}

// RegisterWorker 启动时向控制面注册 worker 信息
// overwrite=false 表示 create-only（推荐默认）
func (r Registrar) RegisterWorker(ctx context.Context, config WorkerConfig, overwrite bool) error {
	if !r.enabled() {
		return nil
	}

	body := map[string]any{
		"worker_name":         config.WorkerName,
		"base_url":            config.BaseURL,
		"redis_addr":          config.RedisAddr,
		"queue_groups":        config.QueueGroups,
		"default_retry_count": config.DefaultRetryCount,
		"default_timeout":     config.DefaultTimeout,
		"default_delay":       config.DefaultDelay,
		"overwrite":           overwrite,
	}
	b, _ := json.Marshal(body)
	u := fmt.Sprintf("%s/api/v1/workers/register", r.ControlPlaneURL)
	req, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("register worker failed: status=%d", resp.StatusCode)
	}
	return nil
}
