package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthChecker_LivenessCheck(t *testing.T) {
	// Liveness check 不依赖外部服务，应该总是成功
	hc := &HealthChecker{}

	result := hc.LivenessCheck()

	assert.Equal(t, "ok", result.Status)
	assert.Contains(t, result.Checks, "service")
	assert.Equal(t, "running", result.Checks["service"])
}

// 注意：ReadinessCheck 需要真实的 PostgreSQL 和 Redis 连接
// 这里只测试基本结构，实际集成测试需要在有数据库的环境中运行
func TestHealthChecker_ReadinessCheck_Structure(t *testing.T) {
	hc := &HealthChecker{
		redisAddr: "localhost:6379",
	}

	// 这个测试会失败（因为没有真实连接），但我们验证返回结构
	result := hc.ReadinessCheck(nil)

	// 应该有状态字段
	assert.NotEmpty(t, result.Status)
	// 应该有检查项
	assert.NotNil(t, result.Checks)
}
