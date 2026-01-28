package dto

import (
	"time"

	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// WorkerListResponse Worker 列表响应
type WorkerListResponse struct {
	Items []workers.Config `json:"items"`
}

// WorkerResponse Worker 详情响应
type WorkerResponse struct {
	Worker workers.Config `json:"worker"`
}

// WorkerStatsResponse Worker 统计响应
type WorkerStatsResponse struct {
	Stats interface{} `json:"stats"`
}

// WorkerTimeSeriesResponse Worker 时间序列统计响应
type WorkerTimeSeriesResponse struct {
	TimeSeries interface{} `json:"timeseries"`
}

// CreateWorkerRequest 创建 Worker 请求
type CreateWorkerRequest struct {
	WorkerName        string         `json:"worker_name" binding:"required" example:"my-worker"`
	BaseURL           string         `json:"base_url" example:"http://localhost:8080"`
	RedisAddr         string         `json:"redis_addr" example:"redis://localhost:6379/0"`
	Concurrency       int32          `json:"concurrency" binding:"required,min=1" example:"10"`
	Queues            map[string]int `json:"queues" binding:"required"`
	DefaultRetryCount int32          `json:"default_retry_count" example:"3"`
	DefaultTimeout    int32          `json:"default_timeout" example:"30"`
	DefaultDelay      int32          `json:"default_delay" example:"0"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Status      string    `json:"status" example:"ok"`
	WorkerName  string    `json:"worker_name" example:"my-worker"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
}

// RegisterWorkerRequest 注册 Worker 请求（与 CreateWorkerRequest 相同）
type RegisterWorkerRequest = CreateWorkerRequest

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"错误信息"`
}
