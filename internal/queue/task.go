package asynqx

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// NewTaskID 生成一个随机 task_id（更符合直觉：每次入队得到一个唯一 ID）。
// 说明：使用 16 字节随机数编码为 hex（32 字符）。
func NewTaskID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type EnqueueParams struct {
	TaskType       string
	TaskKey        string
	Queue          string
	MaxRetry       int32
	TimeoutSeconds int32
	DelaySeconds   int32
	RunAt          time.Time
	Payload        json.RawMessage
}

func EnqueueOptions(p EnqueueParams) []asynq.Option {
	var opts []asynq.Option

	if p.Queue != "" {
		opts = append(opts, asynq.Queue(p.Queue))
	}
	if p.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(int(p.MaxRetry)))
	}
	if p.TimeoutSeconds > 0 {
		opts = append(opts, asynq.Timeout(time.Duration(p.TimeoutSeconds)*time.Second))
	}
	if p.DelaySeconds > 0 {
		opts = append(opts, asynq.ProcessIn(time.Duration(p.DelaySeconds)*time.Second))
	}
	if !p.RunAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(p.RunAt))
	}

	// 幂等：同一个 task_key 24h 内只允许一次（避免误触发重复入队）
	if p.TaskKey != "" {
		opts = append(opts, asynq.Unique(24*time.Hour))
	}

	return opts
}
