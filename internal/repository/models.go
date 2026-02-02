package repository

import (
	"encoding/json"
	"time"
)

// TaskModel GORM 模型 - 对应 task 表
type TaskModel struct {
	ID             int64           `gorm:"primaryKey;autoIncrement;column:id"`
	TaskID         string          `gorm:"column:task_id;uniqueIndex;type:text;not null"`
	WorkerName     string          `gorm:"column:worker_name;type:text;not null;index:idx_task_worker_created_at"`
	Queue          string          `gorm:"column:queue;type:text;not null;index:idx_task_queue_updated_at"`
	Priority       int             `gorm:"column:priority;default:0"`
	Payload        json.RawMessage `gorm:"column:payload;type:jsonb;not null"`
	Status         string          `gorm:"column:status;type:text;not null;index:idx_task_status_updated_at"`
	LastAttempt    int             `gorm:"column:last_attempt;default:0"`
	LastError      *string         `gorm:"column:last_error;type:text"`
	LastWorkerName *string         `gorm:"column:last_worker_name;type:text"`
	TraceID        *string         `gorm:"column:trace_id;type:text"`
	CreatedAt      time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_task_worker_created_at,sort:desc"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;autoUpdateTime;index:idx_task_status_updated_at,sort:desc;index:idx_task_queue_updated_at,sort:desc"`
}

// TableName 指定表名
func (TaskModel) TableName() string { return "task" }

// ToTask 转换为 Task 实体
func (m *TaskModel) ToTask() Task {
	t := Task{
		TaskID:      m.TaskID,
		WorkerName:  m.WorkerName,
		Queue:       m.Queue,
		Priority:    m.Priority,
		Payload:     m.Payload,
		Status:      m.Status,
		LastAttempt: m.LastAttempt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.LastError != nil {
		t.LastError = *m.LastError
	}
	if m.LastWorkerName != nil {
		t.LastWorkerName = *m.LastWorkerName
	}
	if m.TraceID != nil {
		t.TraceID = *m.TraceID
	}
	return t
}

// TaskFromModel 从 Task 实体创建模型
func TaskToModel(t Task) TaskModel {
	m := TaskModel{
		TaskID:      t.TaskID,
		WorkerName:  t.WorkerName,
		Queue:       t.Queue,
		Priority:    t.Priority,
		Payload:     t.Payload,
		Status:      t.Status,
		LastAttempt: t.LastAttempt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.LastError != "" {
		m.LastError = &t.LastError
	}
	if t.LastWorkerName != "" {
		m.LastWorkerName = &t.LastWorkerName
	}
	if t.TraceID != "" {
		m.TraceID = &t.TraceID
	}
	return m
}

// WorkerModel GORM 模型 - 对应 worker 表
type WorkerModel struct {
	ID                int64           `gorm:"primaryKey;autoIncrement;column:id"`
	WorkerName        string          `gorm:"column:worker_name;uniqueIndex;type:text;not null"`
	BaseURL           *string         `gorm:"column:base_url;type:text"`
	RedisAddr         *string         `gorm:"column:redis_addr;type:text"`
	QueueGroups       json.RawMessage `gorm:"column:queue_groups;type:jsonb;not null"`
	DefaultRetryCount int32           `gorm:"column:default_retry_count;default:3"`
	DefaultTimeout    int32           `gorm:"column:default_timeout;default:30"`
	DefaultDelay      int32           `gorm:"column:default_delay;default:0"`
	IsEnabled         bool            `gorm:"column:is_enabled;default:true;index:idx_worker_enabled"`
	LastHeartbeatAt   *time.Time      `gorm:"column:last_heartbeat_at;index:idx_worker_heartbeat"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定表名
func (WorkerModel) TableName() string { return "worker" }

// ToWorkerConfig 转换为 WorkerConfig 实体
func (m *WorkerModel) ToWorkerConfig() WorkerConfig {
	c := WorkerConfig{
		WorkerName:        m.WorkerName,
		DefaultRetryCount: m.DefaultRetryCount,
		DefaultTimeout:    m.DefaultTimeout,
		DefaultDelay:      m.DefaultDelay,
		IsEnabled:         m.IsEnabled,
		LastHeartbeatAt:   m.LastHeartbeatAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
	if m.BaseURL != nil {
		c.BaseURL = *m.BaseURL
	}
	if m.RedisAddr != nil {
		c.RedisAddr = *m.RedisAddr
	}
	// 解析 QueueGroups JSON
	if m.QueueGroups != nil {
		_ = json.Unmarshal(m.QueueGroups, &c.QueueGroups)
	}
	return c
}

// WorkerConfigToModel 从 WorkerConfig 实体创建模型
func WorkerConfigToModel(c WorkerConfig) WorkerModel {
	m := WorkerModel{
		WorkerName:        c.WorkerName,
		DefaultRetryCount: c.DefaultRetryCount,
		DefaultTimeout:    c.DefaultTimeout,
		DefaultDelay:      c.DefaultDelay,
		IsEnabled:         c.IsEnabled,
		LastHeartbeatAt:   c.LastHeartbeatAt,
	}
	if c.BaseURL != "" {
		m.BaseURL = &c.BaseURL
	}
	if c.RedisAddr != "" {
		m.RedisAddr = &c.RedisAddr
	}
	// 序列化 QueueGroups 为 JSON
	if len(c.QueueGroups) > 0 {
		m.QueueGroups, _ = json.Marshal(c.QueueGroups)
	} else {
		m.QueueGroups = []byte("[]")
	}
	return m
}

// TaskAttemptModel GORM 模型 - 对应 task_attempt 表
type TaskAttemptModel struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id"`
	TaskID      string     `gorm:"column:task_id;type:text;not null;index:idx_attempt_task_key_started_at"`
	AsynqTaskID *string    `gorm:"column:asynq_task_id;type:text"`
	Attempt     int        `gorm:"column:attempt;not null"`
	Status      string     `gorm:"column:status;type:text;not null;index:idx_attempt_status_started_at"`
	StartedAt   time.Time  `gorm:"column:started_at;not null;index:idx_attempt_task_key_started_at,sort:desc;index:idx_attempt_status_started_at,sort:desc"`
	FinishedAt  *time.Time `gorm:"column:finished_at"`
	DurationMs  *int       `gorm:"column:duration_ms"`
	Error       *string    `gorm:"column:error;type:text"`
	WorkerName  *string    `gorm:"column:worker_name;type:text"`
	TraceID     *string    `gorm:"column:trace_id;type:text"`
	SpanID      *string    `gorm:"column:span_id;type:text"`
}

// TableName 指定表名
func (TaskAttemptModel) TableName() string { return "task_attempt" }

// ToAttempt 转换为 Attempt 实体
func (m *TaskAttemptModel) ToAttempt() Attempt {
	a := Attempt{
		TaskID:     m.TaskID,
		Attempt:    m.Attempt,
		Status:     m.Status,
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
		DurationMs: m.DurationMs,
	}
	if m.AsynqTaskID != nil {
		a.AsynqTaskID = *m.AsynqTaskID
	}
	if m.Error != nil {
		a.Error = *m.Error
	}
	if m.WorkerName != nil {
		a.WorkerName = *m.WorkerName
	}
	if m.TraceID != nil {
		a.TraceID = *m.TraceID
	}
	if m.SpanID != nil {
		a.SpanID = *m.SpanID
	}
	return a
}

// AttemptToModel 从 Attempt 实体创建模型
func AttemptToModel(a Attempt) TaskAttemptModel {
	m := TaskAttemptModel{
		TaskID:     a.TaskID,
		Attempt:    a.Attempt,
		Status:     a.Status,
		StartedAt:  a.StartedAt,
		FinishedAt: a.FinishedAt,
		DurationMs: a.DurationMs,
	}
	if a.AsynqTaskID != "" {
		m.AsynqTaskID = &a.AsynqTaskID
	}
	if a.Error != "" {
		m.Error = &a.Error
	}
	if a.WorkerName != "" {
		m.WorkerName = &a.WorkerName
	}
	if a.TraceID != "" {
		m.TraceID = &a.TraceID
	}
	if a.SpanID != "" {
		m.SpanID = &a.SpanID
	}
	return m
}
