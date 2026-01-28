package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/azhengyongqin/taskpm/internal/healthcheck"
)

// HealthHandler 健康检查 Handler
type HealthHandler struct {
	healthChecker *healthcheck.HealthChecker
}

// NewHealthHandler 创建 HealthHandler
func NewHealthHandler(healthChecker *healthcheck.HealthChecker) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
	}
}

// Liveness godoc
// @Summary Liveness 检查
// @Description 服务存活检查，用于 Kubernetes liveness probe
// @Tags Health
// @Produce json
// @Success 200 {object} healthcheck.CheckResult
// @Router /healthz [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	if h.healthChecker == nil {
		c.String(http.StatusOK, "ok")
		return
	}
	result := h.healthChecker.LivenessCheck()
	c.JSON(http.StatusOK, result)
}

// Readiness godoc
// @Summary Readiness 检查
// @Description 服务就绪检查，检查依赖服务（PostgreSQL、Redis）状态
// @Tags Health
// @Produce json
// @Success 200 {object} healthcheck.CheckResult
// @Failure 503 {object} healthcheck.CheckResult
// @Router /readyz [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	if h.healthChecker == nil {
		c.String(http.StatusOK, "ok")
		return
	}
	result := h.healthChecker.ReadinessCheck(c.Request.Context())
	if result.Status == "error" {
		c.JSON(http.StatusServiceUnavailable, result)
		return
	}
	c.JSON(http.StatusOK, result)
}
