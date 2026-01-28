package repository

import (
	"context"
	"time"
)

// WorkerConfig Worker 配置信息
type WorkerConfig struct {
	WorkerName        string         `json:"worker_name"`
	BaseURL           string         `json:"base_url,omitempty"`
	RedisAddr         string         `json:"redis_addr,omitempty"`
	Concurrency       int32          `json:"concurrency"`
	Queues            map[string]int `json:"queues"` // queue_name -> weight
	DefaultRetryCount int32          `json:"default_retry_count"`
	DefaultTimeout    int32          `json:"default_timeout"` // seconds
	DefaultDelay      int32          `json:"default_delay"`   // seconds
	IsEnabled         bool           `json:"is_enabled"`
	LastHeartbeatAt   *time.Time     `json:"last_heartbeat_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// WorkerRepository Worker 配置仓储接口
// 抽象持久化层，支持未来迁移到 ClickHouse
type WorkerRepository interface {
	// Upsert 创建或更新 Worker 配置
	Upsert(ctx context.Context, worker WorkerConfig) error

	// Get 根据 worker_name 获取 Worker 配置
	Get(ctx context.Context, workerName string) (*WorkerConfig, error)

	// List 查询所有 Worker 配置列表
	List(ctx context.Context) ([]WorkerConfig, error)

	// Delete 删除 Worker 配置
	Delete(ctx context.Context, workerName string) error

	// UpdateHeartbeat 更新 Worker 心跳时间
	UpdateHeartbeat(ctx context.Context, workerName string, heartbeatAt time.Time) error

	// ListOfflineWorkers 查询离线的 Worker 列表（心跳超过指定时间）
	ListOfflineWorkers(ctx context.Context, offlineDuration time.Duration) ([]WorkerConfig, error)
}
