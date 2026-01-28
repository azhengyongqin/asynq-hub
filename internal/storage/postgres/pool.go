package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxConns          int32         // 最大连接数，默认 20
	MinConns          int32         // 最小连接数，默认 5
	MaxConnLifetime   time.Duration // 连接最大生命周期，默认 30分钟
	MaxConnIdleTime   time.Duration // 连接最大空闲时间，默认 5分钟
	HealthCheckPeriod time.Duration // 健康检查周期，默认 1分钟
}

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          getEnvAsInt32("DB_MAX_CONNS", 20),
		MinConns:          getEnvAsInt32("DB_MIN_CONNS", 5),
		MaxConnLifetime:   getEnvAsDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		MaxConnIdleTime:   getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		HealthCheckPeriod: getEnvAsDuration("DB_HEALTH_CHECK_PERIOD", 1*time.Minute),
	}
}

func NewPool(ctx context.Context, dsn string) (*Pool, error) {
	return NewPoolWithConfig(ctx, dsn, DefaultPoolConfig())
}

func NewPoolWithConfig(ctx context.Context, dsn string, poolCfg PoolConfig) (*Pool, error) {
	if err := validatePostgresURI(dsn); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_DSN: %w", err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// 应用连接池配置
	cfg.MaxConns = poolCfg.MaxConns
	cfg.MinConns = poolCfg.MinConns
	cfg.MaxConnLifetime = poolCfg.MaxConnLifetime
	cfg.MaxConnIdleTime = poolCfg.MaxConnIdleTime
	cfg.HealthCheckPeriod = poolCfg.HealthCheckPeriod

	// 连接超时设置
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// 快速连通性检查
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := p.Ping(cctx); err != nil {
		p.Close()
		return nil, err
	}

	return &Pool{Pool: p}, nil
}

// Stats 返回连接池统计信息
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}

// getEnvAsInt32 从环境变量获取 int32 值
func getEnvAsInt32(key string, defaultVal int32) int32 {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 32); err == nil {
			return int32(intVal)
		}
	}
	return defaultVal
}

// getEnvAsDuration 从环境变量获取 Duration 值（秒）
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultVal
}
