package sdk

// TaskStatus 统一状态枚举，避免用户侧写错字符串。
type TaskStatus string

const (
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFail    TaskStatus = "fail"
	TaskStatusDead    TaskStatus = "dead"
)
