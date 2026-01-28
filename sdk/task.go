package sdk

import (
	"encoding/json"
)

// Task：对齐用户侧示例使用方式
// - 外层 JSON 会作为 asynq task payload
// - Payload 是业务 payload（JSON 原文）
type Task struct {
	ID      string          `json:"id,omitempty"`
	TaskID  string          `json:"task_id,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

func (t *Task) UnmarshalPayload(v any) error {
	return json.Unmarshal(t.Payload, v)
}
