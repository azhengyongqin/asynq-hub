package model

// TaskStatus 统一任务状态枚举（用于 API/PG/前端筛选）。
// 约定：
// - pending: 已入队（等待被 worker 消费）
// - running: worker 开始处理
// - success: 成功
// - fail: 本次尝试失败（可能会重试）
// - dead: 超过最大重试或被判定为不可恢复失败
type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFail    TaskStatus = "fail"
	TaskStatusDead    TaskStatus = "dead"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusPending, TaskStatusRunning, TaskStatusSuccess, TaskStatusFail, TaskStatusDead:
		return true
	default:
		return false
	}
}
