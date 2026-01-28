package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// L 全局 logger
	L *zap.Logger
	// S 全局 SugaredLogger（便捷使用）
	S *zap.SugaredLogger
)

// Init 初始化日志器
func Init(production bool) error {
	var cfg zap.Config
	if production {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	L, err = cfg.Build()
	if err != nil {
		return err
	}
	S = L.Sugar()
	return nil
}

// Sync 刷新日志缓冲区
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}

// WithContext 添加上下文字段
func WithContext(fields ...zap.Field) *zap.Logger {
	if L == nil {
		return zap.NewNop()
	}
	return L.With(fields...)
}

// WithRequestID 添加 request_id
func WithRequestID(requestID string) *zap.Logger {
	return WithContext(zap.String("request_id", requestID))
}

// WithWorkerName 添加 worker_name
func WithWorkerName(workerName string) *zap.Logger {
	return WithContext(zap.String("worker_name", workerName))
}

// WithTaskID 添加 task_id
func WithTaskID(taskID string) *zap.Logger {
	return WithContext(zap.String("task_id", taskID))
}
