package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("POSTGRES_DSN", "postgresql://test:test@localhost:5432/test?sslmode=disable")
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("HTTP_ADDR", ":8080")
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("HTTP_ADDR")
	}()

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, ":8080", cfg.HTTP.Addr)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Contains(t, cfg.Postgres.DSN, "postgresql://")
}

func TestLoadDefaults(t *testing.T) {
	// 只设置必需的配置
	os.Setenv("POSTGRES_DSN", "postgresql://test:test@localhost:5432/test")
	defer os.Unsetenv("POSTGRES_DSN")

	cfg, err := Load()
	require.NoError(t, err)

	// 验证默认值
	assert.Equal(t, ":28080", cfg.HTTP.Addr)
	assert.Equal(t, ":29090", cfg.GRPC.Addr)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, int32(20), cfg.DBPool.MaxConns)
	assert.Equal(t, int32(5), cfg.DBPool.MinConns)
	assert.Equal(t, 30*time.Minute, cfg.DBPool.MaxConnLifetime)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Postgres: PostgresConfig{DSN: "postgresql://localhost/test"},
				Redis:    RedisConfig{Addr: "localhost:6379"},
			},
			wantError: false,
		},
		{
			name: "missing postgres dsn",
			cfg: &Config{
				Redis: RedisConfig{Addr: "localhost:6379"},
			},
			wantError: true,
		},
		{
			name: "missing redis addr",
			cfg: &Config{
				Postgres: PostgresConfig{DSN: "postgresql://localhost/test"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDBPoolConfig(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "postgresql://test:test@localhost:5432/test")
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "10")
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("DB_MIN_CONNS")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, int32(50), cfg.DBPool.MaxConns)
	assert.Equal(t, int32(10), cfg.DBPool.MinConns)
}
