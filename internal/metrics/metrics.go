package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP 请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asynqhub_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "asynqhub_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// 任务指标
	TasksCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asynqhub_tasks_created_total",
			Help: "Total number of tasks created",
		},
		[]string{"worker_name", "queue"},
	)

	TasksCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asynqhub_tasks_completed_total",
			Help: "Total number of tasks completed",
		},
		[]string{"worker_name", "status"},
	)

	TaskExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "asynqhub_task_execution_duration_seconds",
			Help:    "Task execution duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"worker_name", "queue"},
	)

	// 队列指标
	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "asynqhub_queue_size",
			Help: "Number of tasks in queue",
		},
		[]string{"worker_name", "queue", "state"},
	)

	// Worker 指标
	WorkersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "asynqhub_workers_total",
			Help: "Total number of registered workers",
		},
	)

	WorkersOnline = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "asynqhub_workers_online",
			Help: "Number of online workers",
		},
	)

	// 数据库连接池指标
	DBConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "asynqhub_db_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "asynqhub_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	DBConnectionsMax = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "asynqhub_db_connections_max",
			Help: "Maximum number of database connections",
		},
	)

	// 错误指标
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asynqhub_errors_total",
			Help: "Total number of errors",
		},
		[]string{"component", "type"},
	)
)

// RecordHTTPRequest 记录 HTTP 请求
func RecordHTTPRequest(method, path string, status int, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, path, statusClass(status)).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordTaskCreated 记录任务创建
func RecordTaskCreated(workerName, queue string) {
	TasksCreatedTotal.WithLabelValues(workerName, queue).Inc()
}

// RecordTaskCompleted 记录任务完成
func RecordTaskCompleted(workerName, status string, duration float64) {
	TasksCompletedTotal.WithLabelValues(workerName, status).Inc()
	if duration > 0 {
		TaskExecutionDuration.WithLabelValues(workerName, "").Observe(duration)
	}
}

// UpdateQueueSize 更新队列大小
func UpdateQueueSize(workerName, queue, state string, size float64) {
	QueueSize.WithLabelValues(workerName, queue, state).Set(size)
}

// UpdateWorkerStats 更新 Worker 统计
func UpdateWorkerStats(total, online int) {
	WorkersTotal.Set(float64(total))
	WorkersOnline.Set(float64(online))
}

// UpdateDBPoolStats 更新数据库连接池统计
func UpdateDBPoolStats(inUse, idle, max int32) {
	DBConnectionsInUse.Set(float64(inUse))
	DBConnectionsIdle.Set(float64(idle))
	DBConnectionsMax.Set(float64(max))
}

// RecordError 记录错误
func RecordError(component, errorType string) {
	ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// statusClass 将 HTTP 状态码转为类别
func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
