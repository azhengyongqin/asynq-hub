package dto

import (
	"encoding/json"
	"time"
)

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	WorkerName string          `json:"worker_name" binding:"required" example:"my-worker"`
	Queue      string          `json:"queue" binding:"required" example:"web_crawl"` // 队列组名称
	Priority   string          `json:"priority" example:"default"`                   // 优先级：critical, default, low
	Payload    json.RawMessage `json:"payload" binding:"required"`
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	TaskID      string `json:"task_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WorkerName  string `json:"worker_name" example:"my-worker"`
	Queue       string `json:"queue" example:"web_crawl"`
	Priority    string `json:"priority" example:"default"`
	AsynqTaskID string `json:"asynq_task_id"`
	Status      string `json:"status" example:"pending"`
}

// TaskListRequest 任务列表查询请求
type TaskListRequest struct {
	WorkerName string `form:"worker_name" example:"my-worker"`
	Status     string `form:"status" example:"pending"`
	Queue      string `form:"queue" example:"web_crawl"`
	Limit      int    `form:"limit" example:"20"`
	Offset     int    `form:"offset" example:"0"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
}

// TaskResponse 任务详情响应
type TaskResponse struct {
	Task interface{} `json:"task"`
}

// ReplayTaskRequest 重放任务请求
type ReplayTaskRequest struct {
	Delay int `json:"delay" example:"0"`
}

// ReplayTaskResponse 重放任务响应
type ReplayTaskResponse struct {
	Status    string `json:"status" example:"ok"`
	NewTaskID string `json:"new_task_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ReportAttemptRequest 上报任务执行请求
type ReportAttemptRequest struct {
	Attempt    int             `json:"attempt" binding:"required" example:"1"`
	Status     string          `json:"status" binding:"required" example:"success"`
	StartedAt  time.Time       `json:"started_at" binding:"required"`
	FinishedAt *time.Time      `json:"finished_at"`
	Error      string          `json:"error"`
	TraceID    string          `json:"trace_id" example:"trace-123"`
	SpanID     string          `json:"span_id" example:"span-456"`
	Payload    json.RawMessage `json:"payload"`
}

// BatchRetryRequest 批量重试请求
type BatchRetryRequest struct {
	WorkerName string   `json:"worker_name" example:"my-worker"`
	Status     string   `json:"status" example:"fail"`
	TaskIDs    []string `json:"task_ids"`
	Limit      int      `json:"limit" example:"100"`
}

// BatchRetryResponse 批量重试响应
type BatchRetryResponse struct {
	Status       string   `json:"status" example:"ok"`
	TotalRetried int      `json:"total_retried" example:"10"`
	NewTaskIDs   []string `json:"new_task_ids"`
}
