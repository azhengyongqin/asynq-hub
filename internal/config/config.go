package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	HTTP       HTTPConfig
	GRPC       GRPCConfig
	Redis      RedisConfig
	Postgres   PostgresConfig
	DBPool     DBPoolConfig
	Asynq      AsynqConfig
	Monitoring MonitoringConfig
}

// HTTPConfig HTTP 服务配置
type HTTPConfig struct {
	Addr string
}

// GRPCConfig gRPC 服务配置
type GRPCConfig struct {
	Addr string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	DSN string
}

// DBPoolConfig 数据库连接池配置
type DBPoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// AsynqConfig Asynq 配置
type AsynqConfig struct {
	RedisAddr string
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled bool
	Port    int
}

// Load 加载配置
func Load() (*Config, error) {
	v := viper.New()

	// 设置配置文件名和路径
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AddConfigPath("../..")

	// 允许从环境变量读取（优先级最高）
	v.AutomaticEnv()

	// 读取配置文件（如果存在）
	_ = v.ReadInConfig() // 忽略错误，因为可能只使用环境变量

	cfg := &Config{}

	// HTTP 配置
	cfg.HTTP.Addr = v.GetString("HTTP_ADDR")
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":28080"
	}

	// gRPC 配置
	cfg.GRPC.Addr = v.GetString("GRPC_ADDR")
	if cfg.GRPC.Addr == "" {
		cfg.GRPC.Addr = ":29090"
	}

	// Redis 配置
	cfg.Redis.Addr = v.GetString("REDIS_ADDR")
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	cfg.Redis.Password = v.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = v.GetInt("REDIS_DB")

	// PostgreSQL 配置
	cfg.Postgres.DSN = v.GetString("POSTGRES_DSN")
	if cfg.Postgres.DSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	// 数据库连接池配置
	cfg.DBPool.MaxConns = int32(v.GetInt("DB_MAX_CONNS"))
	if cfg.DBPool.MaxConns == 0 {
		cfg.DBPool.MaxConns = 20
	}

	cfg.DBPool.MinConns = int32(v.GetInt("DB_MIN_CONNS"))
	if cfg.DBPool.MinConns == 0 {
		cfg.DBPool.MinConns = 5
	}

	cfg.DBPool.MaxConnLifetime = v.GetDuration("DB_MAX_CONN_LIFETIME")
	if cfg.DBPool.MaxConnLifetime == 0 {
		cfg.DBPool.MaxConnLifetime = 30 * time.Minute
	}

	cfg.DBPool.MaxConnIdleTime = v.GetDuration("DB_MAX_CONN_IDLE_TIME")
	if cfg.DBPool.MaxConnIdleTime == 0 {
		cfg.DBPool.MaxConnIdleTime = 5 * time.Minute
	}

	cfg.DBPool.HealthCheckPeriod = v.GetDuration("DB_HEALTH_CHECK_PERIOD")
	if cfg.DBPool.HealthCheckPeriod == 0 {
		cfg.DBPool.HealthCheckPeriod = 1 * time.Minute
	}

	// Asynq 配置
	cfg.Asynq.RedisAddr = cfg.Redis.Addr

	// 监控配置
	cfg.Monitoring.Enabled = v.GetBool("MONITORING_ENABLED")
	cfg.Monitoring.Port = v.GetInt("MONITORING_PORT")
	if cfg.Monitoring.Port == 0 {
		cfg.Monitoring.Port = 29091
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Postgres.DSN == "" {
		return fmt.Errorf("PostgreSQL DSN is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("Redis address is required")
	}
	return nil
}
