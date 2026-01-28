package sdk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HeartbeatManager 心跳管理器
type HeartbeatManager struct {
	workerName      string
	controlPlaneURL string
	interval        time.Duration
	timeout         time.Duration

	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
}

// NewHeartbeatManager 创建心跳管理器
func NewHeartbeatManager(workerName, controlPlaneURL string) *HeartbeatManager {
	return &HeartbeatManager{
		workerName:      workerName,
		controlPlaneURL: controlPlaneURL,
		interval:        30 * time.Second, // 默认30秒一次心跳
		timeout:         5 * time.Second,  // 心跳请求超时时间
		stopCh:          make(chan struct{}),
	}
}

// SetInterval 设置心跳间隔
func (h *HeartbeatManager) SetInterval(interval time.Duration) {
	h.interval = interval
}

// Start 启动心跳
func (h *HeartbeatManager) Start(ctx context.Context) {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// 立即发送一次心跳
	h.sendHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.sendHeartbeat(ctx)
		}
	}
}

// Stop 停止心跳
func (h *HeartbeatManager) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.stopped {
		close(h.stopCh)
		h.stopped = true
	}
}

// sendHeartbeat 发送心跳到控制面
func (h *HeartbeatManager) sendHeartbeat(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	client := NewClient(h.controlPlaneURL)

	err := client.UpdateHeartbeat(ctx, h.workerName)
	if err != nil {
		log.Printf("[heartbeat] 发送心跳失败: %v", err)
		return
	}

	log.Printf("[heartbeat] 心跳发送成功: worker=%s", h.workerName)
}

// ReportRetryConfig 上报重试配置
type ReportRetryConfig struct {
	MaxRetries     int           // 最大重试次数，默认 3
	InitialBackoff time.Duration // 初始退避时间，默认 1秒
	MaxBackoff     time.Duration // 最大退避时间，默认 30秒
	BackoffFactor  float64       // 退避因子，默认 2.0（指数退避）
}

// DefaultReportRetryConfig 默认重试配置
func DefaultReportRetryConfig() ReportRetryConfig {
	return ReportRetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
}

// ReportWithRetry 带重试的状态上报
func ReportWithRetry(ctx context.Context, reporter Reporter, taskID string, req ReportAttemptRequest, config ReportRetryConfig) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待退避时间
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			// 计算下次退避时间（指数退避）
			backoff = time.Duration(float64(backoff) * config.BackoffFactor)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		err := reporter.ReportAttempt(ctx, taskID, req)
		if err == nil {
			if attempt > 0 {
				log.Printf("[report-retry] 重试成功: task_id=%s, attempt=%d", taskID, attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("[report-retry] 上报失败 (尝试 %d/%d): %v", attempt+1, config.MaxRetries+1, err)
	}

	return fmt.Errorf("上报失败，已达最大重试次数: %w", lastErr)
}

// GracefulShutdownManager 优雅关闭管理器
type GracefulShutdownManager struct {
	timeout       time.Duration
	shutdownHooks []func(context.Context) error
	mu            sync.Mutex
}

// NewGracefulShutdownManager 创建优雅关闭管理器
func NewGracefulShutdownManager(timeout time.Duration) *GracefulShutdownManager {
	return &GracefulShutdownManager{
		timeout:       timeout,
		shutdownHooks: make([]func(context.Context) error, 0),
	}
}

// AddHook 添加关闭钩子
func (g *GracefulShutdownManager) AddHook(hook func(context.Context) error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.shutdownHooks = append(g.shutdownHooks, hook)
}

// Shutdown 执行优雅关闭
func (g *GracefulShutdownManager) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	g.mu.Lock()
	hooks := make([]func(context.Context) error, len(g.shutdownHooks))
	copy(hooks, g.shutdownHooks)
	g.mu.Unlock()

	log.Printf("[shutdown] 开始优雅关闭，超时时间: %v", g.timeout)

	// 执行所有钩子
	for i, hook := range hooks {
		if err := hook(ctx); err != nil {
			log.Printf("[shutdown] 钩子 %d 执行失败: %v", i, err)
			return err
		}
	}

	log.Printf("[shutdown] 优雅关闭完成")
	return nil
}

// TaskContext 任务执行上下文（带超时控制）
type TaskContext struct {
	context.Context
	TaskID  string
	Attempt int
	Timeout time.Duration
}

// NewTaskContext 创建任务上下文
func NewTaskContext(parent context.Context, taskID string, attempt int, timeout time.Duration) (TaskContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	return TaskContext{
		Context: ctx,
		TaskID:  taskID,
		Attempt: attempt,
		Timeout: timeout,
	}, cancel
}

// IsTimeout 判断是否超时
func (tc TaskContext) IsTimeout() bool {
	select {
	case <-tc.Done():
		return tc.Err() == context.DeadlineExceeded
	default:
		return false
	}
}

// RemainingTime 获取剩余时间
func (tc TaskContext) RemainingTime() time.Duration {
	deadline, ok := tc.Deadline()
	if !ok {
		return 0
	}
	return time.Until(deadline)
}
