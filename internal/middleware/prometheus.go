package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/azhengyongqin/asynq-hub/internal/metrics"
)

// PrometheusMiddleware 记录 HTTP 请求指标
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.RecordHTTPRequest(c.Request.Method, path, status, duration)
	}
}

// RequestIDMiddleware 为每个请求生成唯一 ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 使用时间戳 + 随机数生成简单的 request ID
			requestID = strconv.FormatInt(time.Now().UnixNano(), 36)
		}

		// 设置到 context 和响应头
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}
