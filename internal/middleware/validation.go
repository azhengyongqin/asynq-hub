package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// MaxPayloadSize 最大 payload 大小（2MB）
	MaxPayloadSize = 2 * 1024 * 1024
)

var (
	// WorkerNameRegex Worker 名称正则（字母数字下划线连字符，3-64字符）
	WorkerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

	// QueueNameRegex 队列名称正则（字母数字下划线连字符，1-64字符）
	QueueNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

	// TaskIDRegex TaskID 正则（字母数字连字符，1-128字符）
	TaskIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{1,128}$`)
)

// PayloadSizeLimit Payload 大小限制中间件
func PayloadSizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "请求体过大，最大允许 2MB",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidateWorkerName 验证 Worker 名称
func ValidateWorkerName(workerName string) bool {
	return WorkerNameRegex.MatchString(workerName)
}

// ValidateQueueName 验证队列名称
func ValidateQueueName(queueName string) bool {
	return QueueNameRegex.MatchString(queueName)
}

// ValidateTaskID 验证 Task ID
func ValidateTaskID(taskID string) bool {
	return TaskIDRegex.MatchString(taskID)
}

// SanitizeString 清理字符串（去除危险字符）
func SanitizeString(s string) string {
	// 去除前后空格
	s = strings.TrimSpace(s)

	// 去除控制字符
	var builder strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// ValidateWorkerNameParam Gin 中间件：验证路径参数中的 worker_name
func ValidateWorkerNameParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		workerName := c.Param("worker_name")
		if workerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "worker_name 参数缺失",
			})
			c.Abort()
			return
		}

		if !ValidateWorkerName(workerName) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "worker_name 格式无效，必须是3-64个字母、数字、下划线或连字符",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateTaskIDParam Gin 中间件：验证路径参数中的 task_id
func ValidateTaskIDParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("task_id")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "task_id 参数缺失",
			})
			c.Abort()
			return
		}

		if !ValidateTaskID(taskID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "task_id 格式无效，必须是1-128个字母、数字或连字符",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitSimple 简单的速率限制（基于 IP）
// 注意：生产环境建议使用更完善的限流方案（如 Redis + Token Bucket）
func RateLimitSimple(requestsPerMinute int) gin.HandlerFunc {
	// TODO: 实现简单的内存限流
	// 生产环境建议使用 github.com/ulule/limiter 或类似库
	return func(c *gin.Context) {
		// 暂时跳过，待实现
		c.Next()
	}
}

// CORSMiddleware CORS 中间件（内部系统可选）
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
