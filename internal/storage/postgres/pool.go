package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB GORM 数据库封装
type DB struct {
	*gorm.DB
}

// DBConfig 数据库连接配置
type DBConfig struct {
	MaxOpenConns    int           // 最大打开连接数，默认 20
	MaxIdleConns    int           // 最大空闲连接数，默认 5
	ConnMaxLifetime time.Duration // 连接最大生命周期，默认 30分钟
	ConnMaxIdleTime time.Duration // 连接最大空闲时间，默认 5分钟
}

// DefaultDBConfig 返回默认数据库配置
func DefaultDBConfig() DBConfig {
	return DBConfig{
		MaxOpenConns:    getEnvAsInt("DB_MAX_CONNS", 20),
		MaxIdleConns:    getEnvAsInt("DB_MIN_CONNS", 5),
		ConnMaxLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		ConnMaxIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
	}
}

// NewDB 使用默认配置创建数据库连接
func NewDB(ctx context.Context, dsn string) (*DB, error) {
	return NewDBWithConfig(ctx, dsn, DefaultDBConfig())
}

// NewDBWithConfig 使用指定配置创建数据库连接
func NewDBWithConfig(ctx context.Context, dsn string, cfg DBConfig) (*DB, error) {
	if err := validatePostgresURI(dsn); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_DSN: %w", err)
	}

	// 创建 GORM 实例
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 获取底层 sql.DB 并配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 连通性检查
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close 关闭数据库连接
func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SqlDB 返回底层 sql.DB（用于健康检查等）
func (d *DB) SqlDB() (*sql.DB, error) {
	return d.DB.DB()
}

// getEnvAsInt 从环境变量获取 int 值
func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
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
