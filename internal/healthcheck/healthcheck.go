package healthcheck

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	pgPool      *pgxpool.Pool
	asynqClient *asynq.Client
	redisAddr   string
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(pgPool *pgxpool.Pool, asynqClient *asynq.Client, redisAddr string) *HealthChecker {
	return &HealthChecker{
		pgPool:      pgPool,
		asynqClient: asynqClient,
		redisAddr:   redisAddr,
	}
}

// CheckResult 健康检查结果
type CheckResult struct {
	Status  string            `json:"status"` // "ok" or "error"
	Checks  map[string]string `json:"checks"`
	Version string            `json:"version,omitempty"`
}

// LivenessCheck 存活检查（快速返回，不检查依赖）
func (h *HealthChecker) LivenessCheck() CheckResult {
	return CheckResult{
		Status: "ok",
		Checks: map[string]string{
			"service": "running",
		},
	}
}

// ReadinessCheck 就绪检查（检查所有依赖）
func (h *HealthChecker) ReadinessCheck(ctx context.Context) CheckResult {
	result := CheckResult{
		Checks: make(map[string]string),
	}

	// 检查 PostgreSQL 连接
	if h.pgPool != nil {
		if err := h.checkPostgres(ctx); err != nil {
			result.Checks["postgres"] = "error: " + err.Error()
			result.Status = "error"
		} else {
			result.Checks["postgres"] = "ok"
		}
	}

	// 检查 Redis 连接（通过 Asynq）
	if h.asynqClient != nil {
		if err := h.checkRedis(ctx); err != nil {
			result.Checks["redis"] = "error: " + err.Error()
			result.Status = "error"
		} else {
			result.Checks["redis"] = "ok"
		}
	}

	// 如果所有检查都通过
	if result.Status == "" {
		result.Status = "ok"
	}

	return result
}

// checkPostgres 检查 PostgreSQL 连接
func (h *HealthChecker) checkPostgres(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return h.pgPool.Ping(ctx)
}

// checkRedis 检查 Redis 连接（通过 Asynq Inspector）
func (h *HealthChecker) checkRedis(ctx context.Context) error {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: h.redisAddr})
	defer inspector.Close()

	// 尝试获取队列列表
	_, err := inspector.Queues()
	return err
}
