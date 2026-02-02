package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/azhengyongqin/asynq-hub/internal/repository"
	"github.com/azhengyongqin/asynq-hub/internal/server/dto"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// WorkerHandler Worker 相关 API Handler
type WorkerHandler struct {
	workerStore *workers.Store
	workerRepo  repository.WorkerRepository
	taskRepo    repository.TaskRepository
}

// NewWorkerHandler 创建 WorkerHandler
func NewWorkerHandler(workerStore *workers.Store, workerRepo repository.WorkerRepository, taskRepo repository.TaskRepository) *WorkerHandler {
	return &WorkerHandler{
		workerStore: workerStore,
		workerRepo:  workerRepo,
		taskRepo:    taskRepo,
	}
}

// ListWorkers godoc
// @Summary 获取 Worker 列表
// @Description 获取所有已注册的 Worker 配置列表
// @Tags Workers
// @Produce json
// @Success 200 {object} dto.WorkerListResponse
// @Router /api/v1/workers [get]
func (h *WorkerHandler) ListWorkers(c *gin.Context) {
	c.JSON(http.StatusOK, dto.WorkerListResponse{
		Items: h.workerStore.List(),
	})
}

// GetWorker godoc
// @Summary 获取 Worker 详情
// @Description 根据 worker_name 获取 Worker 配置详情
// @Tags Workers
// @Produce json
// @Param worker_name path string true "Worker 名称"
// @Success 200 {object} dto.WorkerResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/workers/{worker_name} [get]
func (h *WorkerHandler) GetWorker(c *gin.Context) {
	workerName := c.Param("worker_name")
	item, ok := h.workerStore.Get(workerName)
	if !ok {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}
	c.JSON(http.StatusOK, dto.WorkerResponse{Worker: item})
}

// GetWorkerStats godoc
// @Summary 获取 Worker 统计信息
// @Description 获取指定 Worker 的任务统计信息
// @Tags Workers
// @Produce json
// @Param worker_name path string true "Worker 名称"
// @Success 200 {object} dto.WorkerStatsResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 501 {object} dto.ErrorResponse
// @Router /api/v1/workers/{worker_name}/stats [get]
func (h *WorkerHandler) GetWorkerStats(c *gin.Context) {
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	workerName := c.Param("worker_name")
	stats, err := h.taskRepo.GetWorkerStats(c.Request.Context(), workerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.WorkerStatsResponse{Stats: stats})
}

// GetWorkerTimeSeries godoc
// @Summary 获取 Worker 时间序列统计
// @Description 获取指定 Worker 的时间序列统计数据
// @Tags Workers
// @Produce json
// @Param worker_name path string true "Worker 名称"
// @Param hours query int false "统计小时数" default(24)
// @Success 200 {object} dto.WorkerTimeSeriesResponse
// @Failure 501 {object} dto.ErrorResponse
// @Router /api/v1/workers/{worker_name}/timeseries [get]
func (h *WorkerHandler) GetWorkerTimeSeries(c *gin.Context) {
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	workerName := c.Param("worker_name")
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	timeseries, err := h.taskRepo.GetWorkerTimeSeriesStats(c.Request.Context(), workerName, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.WorkerTimeSeriesResponse{TimeSeries: timeseries})
}

// CreateOrUpdateWorker godoc
// @Summary 创建或更新 Worker
// @Description 创建新的 Worker 或更新已存在的 Worker 配置
// @Tags Workers
// @Accept json
// @Produce json
// @Param request body dto.CreateWorkerRequest true "Worker 配置"
// @Success 200 {object} dto.WorkerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/workers [post]
func (h *WorkerHandler) CreateOrUpdateWorker(c *gin.Context) {
	var req dto.CreateWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 转换队列组配置
	queueGroups := make([]workers.QueueGroupConfig, len(req.QueueGroups))
	for i, qg := range req.QueueGroups {
		queueGroups[i] = workers.QueueGroupConfig{
			Name:        qg.Name,
			Concurrency: qg.Concurrency,
			Priorities:  qg.Priorities,
		}
	}

	config := workers.Config{
		WorkerName:        req.WorkerName,
		BaseURL:           req.BaseURL,
		RedisAddr:         req.RedisAddr,
		QueueGroups:       queueGroups,
		DefaultRetryCount: req.DefaultRetryCount,
		DefaultTimeout:    req.DefaultTimeout,
		DefaultDelay:      req.DefaultDelay,
	}

	item, err := h.workerStore.Upsert(config)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if h.workerRepo != nil {
		repoConfig := repository.WorkerConfig{
			WorkerName:        item.WorkerName,
			BaseURL:           item.BaseURL,
			RedisAddr:         item.RedisAddr,
			QueueGroups:       convertToRepoQueueGroups(item.QueueGroups),
			DefaultRetryCount: item.DefaultRetryCount,
			DefaultTimeout:    item.DefaultTimeout,
			DefaultDelay:      item.DefaultDelay,
			IsEnabled:         item.IsEnabled,
			LastHeartbeatAt:   item.LastHeartbeatAt,
		}
		if err := h.workerRepo.Upsert(c.Request.Context(), repoConfig); err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, dto.WorkerResponse{Worker: item})
}

// UpdateHeartbeat godoc
// @Summary 更新 Worker 心跳
// @Description 更新指定 Worker 的心跳时间
// @Tags Workers
// @Produce json
// @Param worker_name path string true "Worker 名称"
// @Success 200 {object} dto.HeartbeatResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/workers/{worker_name}/heartbeat [post]
func (h *WorkerHandler) UpdateHeartbeat(c *gin.Context) {
	workerName := c.Param("worker_name")

	// 检查 worker 是否存在
	config, ok := h.workerStore.Get(workerName)
	if !ok {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}

	// 更新心跳时间
	now := time.Now()
	config.LastHeartbeatAt = &now

	// 更新内存存储
	_, err := h.workerStore.Upsert(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 更新数据库
	if h.workerRepo != nil {
		if err := h.workerRepo.UpdateHeartbeat(c.Request.Context(), workerName, now); err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, dto.HeartbeatResponse{
		Status:      "ok",
		WorkerName:  workerName,
		HeartbeatAt: now,
	})
}

// RegisterWorker godoc
// @Summary 注册 Worker
// @Description Worker SDK 自动注册接口
// @Tags Workers
// @Accept json
// @Produce json
// @Param request body dto.RegisterWorkerRequest true "Worker 注册信息"
// @Success 200 {object} dto.WorkerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/workers/register [post]
func (h *WorkerHandler) RegisterWorker(c *gin.Context) {
	var req struct {
		dto.RegisterWorkerRequest
		Overwrite bool `json:"overwrite"` // true 时覆盖已有配置（谨慎使用）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 检查是否已存在
	_, exists := h.workerStore.Get(req.WorkerName)
	if exists && !req.Overwrite {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "worker 已存在，跳过注册（使用 overwrite=true 强制更新）",
		})
		return
	}

	// 转换队列组配置
	queueGroups := make([]workers.QueueGroupConfig, len(req.QueueGroups))
	for i, qg := range req.QueueGroups {
		queueGroups[i] = workers.QueueGroupConfig{
			Name:        qg.Name,
			Concurrency: qg.Concurrency,
			Priorities:  qg.Priorities,
		}
	}

	now := time.Now()
	config := workers.Config{
		WorkerName:        req.WorkerName,
		BaseURL:           req.BaseURL,
		RedisAddr:         req.RedisAddr,
		QueueGroups:       queueGroups,
		DefaultRetryCount: req.DefaultRetryCount,
		DefaultTimeout:    req.DefaultTimeout,
		DefaultDelay:      req.DefaultDelay,
		IsEnabled:         true,
		LastHeartbeatAt:   &now,
	}

	item, err := h.workerStore.Upsert(config)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if h.workerRepo != nil {
		repoConfig := repository.WorkerConfig{
			WorkerName:        item.WorkerName,
			BaseURL:           item.BaseURL,
			RedisAddr:         item.RedisAddr,
			QueueGroups:       convertToRepoQueueGroups(item.QueueGroups),
			DefaultRetryCount: item.DefaultRetryCount,
			DefaultTimeout:    item.DefaultTimeout,
			DefaultDelay:      item.DefaultDelay,
			IsEnabled:         item.IsEnabled,
			LastHeartbeatAt:   item.LastHeartbeatAt,
		}
		if err := h.workerRepo.Upsert(c.Request.Context(), repoConfig); err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"worker": item,
	})
}

// convertToRepoQueueGroups 将 workers.QueueGroupConfig 转换为 repository.QueueGroupConfig
func convertToRepoQueueGroups(queueGroups []workers.QueueGroupConfig) []repository.QueueGroupConfig {
	result := make([]repository.QueueGroupConfig, len(queueGroups))
	for i, qg := range queueGroups {
		result[i] = repository.QueueGroupConfig{
			Name:        qg.Name,
			Concurrency: qg.Concurrency,
			Priorities:  qg.Priorities,
		}
	}
	return result
}
