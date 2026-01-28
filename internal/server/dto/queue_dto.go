package dto

// QueueStatsRequest 队列统计查询请求
type QueueStatsRequest struct {
	WorkerName string `form:"worker_name" binding:"required" example:"my-worker"`
}

// QueueStatsResponse 队列统计响应
type QueueStatsResponse struct {
	WorkerName string        `json:"worker_name" example:"my-worker"`
	Queues     []interface{} `json:"queues"`
}

// ClearQueueRequest 清空队列请求
type ClearQueueRequest struct {
	WorkerName string `json:"worker_name" binding:"required" example:"my-worker"`
	QueueName  string `json:"queue_name" example:"default"`
}

// ClearQueueResponse 清空队列响应
type ClearQueueResponse struct {
	Status        string   `json:"status" example:"ok"`
	WorkerName    string   `json:"worker_name" example:"my-worker"`
	ClearedQueues []string `json:"cleared_queues"`
	TotalDeleted  int      `json:"total_deleted" example:"100"`
}

// ClearDeadQueueRequest 清空死信队列请求
type ClearDeadQueueRequest struct {
	WorkerName string `json:"worker_name" binding:"required" example:"my-worker"`
	QueueName  string `json:"queue_name" example:"default"`
}

// ClearDeadQueueResponse 清空死信队列响应
type ClearDeadQueueResponse struct {
	Status        string   `json:"status" example:"ok"`
	WorkerName    string   `json:"worker_name" example:"my-worker"`
	ClearedQueues []string `json:"cleared_queues"`
	TotalDeleted  int      `json:"total_deleted" example:"50"`
}
