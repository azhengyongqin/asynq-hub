package repository

import (
	"context"
	"encoding/json"
	"time"
)

// Task 表示任务实体
type Task struct {
	TaskID         string          `json:"task_id"`
	WorkerName     string          `json:"worker_name"`
	Queue          string          `json:"queue"`
	Priority       int             `json:"priority"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	LastAttempt    int             `json:"last_attempt"`
	LastError      string          `json:"last_error,omitempty"`
	LastWorkerName string          `json:"last_worker_name,omitempty"`
	TraceID        string          `json:"trace_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Attempt 表示任务执行尝试记录
type Attempt struct {
	TaskID      string     `json:"task_id"`
	AsynqTaskID string     `json:"asynq_task_id,omitempty"`
	Attempt     int        `json:"attempt"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	DurationMs  *int       `json:"duration_ms,omitempty"`
	Error       string     `json:"error,omitempty"`
	WorkerName  string     `json:"worker_name,omitempty"`
	TraceID     string     `json:"trace_id,omitempty"`
	SpanID      string     `json:"span_id,omitempty"`
}

// ListTasksFilter 任务列表查询过滤条件
type ListTasksFilter struct {
	WorkerName string
	Status     string
	Queue      string
	Limit      int
	Offset     int
}

// WorkerStats Worker 统计信息
type WorkerStats struct {
	TotalTasks        int            `json:"total_tasks"`
	SuccessTasks      int            `json:"success_tasks"`
	FailedTasks       int            `json:"failed_tasks"`
	PendingTasks      int            `json:"pending_tasks"`
	RunningTasks      int            `json:"running_tasks"`
	SuccessRate       float64        `json:"success_rate"`
	AvgDurationMs     int            `json:"avg_duration_ms"`
	QueueStats        map[string]int `json:"queue_stats"`
	QueueSuccessStats map[string]int `json:"queue_success_stats"`
	QueueFailedStats  map[string]int `json:"queue_failed_stats"`
	QueueAvgDuration  map[string]int `json:"queue_avg_duration"`
}

// TimeSeriesStats 时间序列统计数据
type TimeSeriesStats struct {
	Hour         string `json:"hour"`
	TotalTasks   int    `json:"total_tasks"`
	SuccessTasks int    `json:"success_tasks"`
	FailedTasks  int    `json:"failed_tasks"`
	AvgDuration  int    `json:"avg_duration"`
}

// TaskRepository 任务仓储接口
// 抽象持久化层，支持未来迁移到 ClickHouse
type TaskRepository interface {
	// UpsertTask 创建或更新任务
	UpsertTask(ctx context.Context, task Task) error

	// UpdateTaskStatus 更新任务状态
	UpdateTaskStatus(ctx context.Context, taskID, status string, lastAttempt int, lastError string, lastWorkerName string) error

	// GetTask 根据 task_id 获取任务详情
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// ListTasks 查询任务列表（支持分页和过滤）
	ListTasks(ctx context.Context, filter ListTasksFilter) ([]Task, error)

	// CountTasks 统计任务总数
	CountTasks(ctx context.Context, filter ListTasksFilter) (int, error)

	// InsertAttempt 插入任务执行尝试记录
	InsertAttempt(ctx context.Context, attempt Attempt) error

	// ListAttempts 查询任务的执行尝试历史
	ListAttempts(ctx context.Context, taskID string, limit int) ([]Attempt, error)

	// GetWorkerStats 获取 Worker 统计信息
	GetWorkerStats(ctx context.Context, workerName string) (*WorkerStats, error)

	// GetWorkerTimeSeriesStats 获取 Worker 时间序列统计数据
	GetWorkerTimeSeriesStats(ctx context.Context, workerName string, hours int) ([]TimeSeriesStats, error)

	// ListFailedTasks 查询失败的任务列表（用于批量重试）
	ListFailedTasks(ctx context.Context, workerName string, limit int) ([]Task, error)
}
