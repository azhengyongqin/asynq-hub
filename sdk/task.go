package sdk

import (
	"encoding/json"
)

// Task：SDK 任务结构体
// - TaskID 是任务的业务唯一标识（对应数据库 task_id 字段）
// - Payload 是业务 payload（JSON 原文）
type Task struct {
	TaskID  string          `json:"task_id,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

func (t *Task) UnmarshalPayload(v any) error {
	return json.Unmarshal(t.Payload, v)
}
