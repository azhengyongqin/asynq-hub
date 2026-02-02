package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/azhengyongqin/asynq-hub/internal/logger"
)

const (
	// MaxBodyLogSize 最大记录的请求/响应体大小（字节）
	MaxBodyLogSize = 4096
)

// responseWriter 是 gin.ResponseWriter 的包装器，用于拦截和记录 HTTP 响应数据
//
// 工作原理：
// 1. Gin 框架在处理请求时，会调用 ResponseWriter.Write() 方法将响应数据写入客户端
// 2. 通过包装原始的 ResponseWriter，我们可以在数据发送给客户端之前拦截它
// 3. 这样就能记录响应体内容和大小，用于日志记录和调试
//
// 使用场景：
// - 记录 API 响应内容，便于调试和审计
// - 统计响应体大小，用于性能分析
// - 在发生 5xx 错误时，记录完整响应体以便排查问题
type responseWriter struct {
	gin.ResponseWriter               // 嵌入原始的 ResponseWriter，保留所有原始功能
	body               *bytes.Buffer // 缓存响应体内容（仅缓存前 4KB，避免内存占用过大）
	size               int           // 记录响应体的总大小（字节数）
}

// Write 实现 io.Writer 接口，拦截响应数据的写入操作
//
// 执行流程：
// 1. 先调用原始 ResponseWriter.Write()，将数据发送给客户端（保证正常响应）
// 2. 累加响应体的总大小（用于统计）
// 3. 如果响应体较小（≤4KB），则缓存到 body 中（用于日志记录）
// 4. 如果响应体过大，则不再缓存（避免内存占用过大）
//
// 参数：
//
//	b - 要写入的响应数据
//
// 返回值：
//
//	size - 实际写入的字节数
//	err  - 写入过程中的错误（如果有）
func (w *responseWriter) Write(b []byte) (int, error) {
	// 先写入原始 ResponseWriter，确保客户端能正常收到响应
	size, err := w.ResponseWriter.Write(b)

	// 累加响应体总大小（无论是否缓存，都要统计）
	w.size += size

	// 只在响应体较小时才缓存到内存中
	// 这样可以在日志中记录响应内容，同时避免大响应体占用过多内存
	// 例如：JSON 错误信息通常很小，适合缓存；但文件下载响应可能很大，不应缓存
	if w.body.Len()+len(b) <= MaxBodyLogSize {
		w.body.Write(b)
	}

	return size, err
}

// LoggingMiddleware 记录请求日志
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 获取 request_id（由 RequestIDMiddleware 设置）
		requestID, _ := c.Get("request_id")

		// 获取路径（优先使用路由模板）
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 读取请求体（仅对 POST/PUT/PATCH 且体积较小时记录）
		var requestBody string
		if c.Request.Body != nil && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// 恢复请求体，以便后续处理器可以读取
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// 只记录较小的请求体
				if len(bodyBytes) > 0 && len(bodyBytes) <= MaxBodyLogSize {
					requestBody = string(bodyBytes)
				} else if len(bodyBytes) > MaxBodyLogSize {
					requestBody = string(bodyBytes[:MaxBodyLogSize]) + "... (truncated)"
				}
			}
		}

		// 包装 ResponseWriter 以捕获响应
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
			size:           0,
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算请求耗时
		duration := time.Since(start)
		status := c.Writer.Status()

		// 构建日志事件
		var logEvent *zerolog.Event
		if status >= 500 {
			// 5xx 错误：Error 级别
			logEvent = logger.L.Error()
		} else if status >= 400 {
			// 4xx 错误：Warn 级别
			logEvent = logger.L.Warn()
		} else {
			// 正常请求：Info 级别
			logEvent = logger.L.Info()
		}

		// 添加通用字段
		if requestID != nil {
			logEvent = logEvent.Interface("request_id", requestID)
		}
		logEvent = logEvent.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("duration(ms)", duration).
			Int("response_size", blw.size).
			Str("client_ip", c.ClientIP())

		// 添加查询参数（如果存在）
		if c.Request.URL.RawQuery != "" {
			logEvent = logEvent.Str("query", c.Request.URL.RawQuery)
		}

		// 添加请求体（如果记录了）
		if requestBody != "" {
			logEvent = logEvent.Str("request_body", requestBody)
		}

		// 记录错误信息（如果有）
		if len(c.Errors) > 0 {
			logEvent = logEvent.Str("errors", c.Errors.String())
		}

		// 对于 5xx 错误，记录响应体以便调试
		if status >= 500 && blw.body.Len() > 0 {
			logEvent = logEvent.Str("response_body", blw.body.String())
		}

		logEvent.Msg("HTTP 请求")
	}
}

// GetRequestID 从上下文中获取请求 ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
