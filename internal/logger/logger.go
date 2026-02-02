package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var (
	// L 全局 logger
	L zerolog.Logger
)

// Init 初始化日志器
func Init(production bool) error {
	// 设置时间格式
	zerolog.TimeFieldFormat = time.RFC3339

	if production {
		// 生产环境：JSON 格式输出
		L = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		// 开发环境：控制台友好格式
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			// 自定义字段输出顺序（HTTP 请求日志的常见顺序）
			FieldsOrder: []string{
				"request_id",    // 1. 请求 ID
				"method",        // 2. HTTP 方法
				"path",          // 3. 请求路径
				"status",        // 4. 状态码
				"duration(ms)",  // 5. 耗时
				"response_size", // 6. 响应大小
				"client_ip",     // 7. 客户端 IP
				"query",         // 8. 查询参数
				"request_body",  // 9. 请求体
				"response_body", // 10. 响应体
				"errors",        // 11. 错误信息
			},
		}
		L = zerolog.New(output).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	// 设置全局日志级别
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	return nil
}

// Sync zerolog 不需要显式 sync，保留接口兼容性
func Sync() {
	// zerolog 不需要显式 sync
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// WithRequestID 添加 request_id
func WithRequestID(requestID string) zerolog.Logger {
	return L.With().Str("request_id", requestID).Logger()
}

// WithWorkerName 添加 worker_name
func WithWorkerName(workerName string) zerolog.Logger {
	return L.With().Str("worker_name", workerName).Logger()
}

// WithTaskID 添加 task_id
func WithTaskID(taskID string) zerolog.Logger {
	return L.With().Str("task_id", taskID).Logger()
}

// Debug 输出 debug 级别日志
func Debug() *zerolog.Event {
	return L.Debug()
}

// Info 输出 info 级别日志
func Info() *zerolog.Event {
	return L.Info()
}

// Warn 输出 warn 级别日志
func Warn() *zerolog.Event {
	return L.Warn()
}

// Error 输出 error 级别日志
func Error() *zerolog.Event {
	return L.Error()
}

// Fatal 输出 fatal 级别日志并退出
func Fatal() *zerolog.Event {
	return L.Fatal()
}
