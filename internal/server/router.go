package httpserver

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/azhengyongqin/asynq-hub/internal/healthcheck"
	"github.com/azhengyongqin/asynq-hub/internal/middleware"
	"github.com/azhengyongqin/asynq-hub/internal/model"
	asynqx "github.com/azhengyongqin/asynq-hub/internal/queue"
	"github.com/azhengyongqin/asynq-hub/internal/repository"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

type Deps struct {
	WorkerStore *workers.Store

	// AsynqClient 用于入队
	AsynqClient *asynq.Client

	// 可选：若提供则会把数据落到 Postgres（用于列表/详情/报表）
	WorkerRepo *repository.WorkerRepo
	TaskRepo   *repository.TaskRepo

	// HealthChecker 健康检查器
	HealthChecker *healthcheck.HealthChecker

	// WebFS 前端静态文件（可选）
	WebFS *embed.FS
}

// NewRouter 提供 Gin HTTP API
// @title Asynq-Hub API
// @version 1.0.0
// @description 分布式任务管理系统 API
// @BasePath /api/v1
// @schemes http https
func NewRouter(deps Deps) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	// 全局中间件
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.PayloadSizeLimit(middleware.MaxPayloadSize))
	r.Use(middleware.CORSMiddleware())

	// @Summary Liveness 检查
	// @Description 服务存活检查，用于 Kubernetes liveness probe
	// @Tags Health
	// @Produce json
	// @Success 200 {object} dto.HealthResponse
	// @Router /healthz [get]
	r.GET("/healthz", func(c *gin.Context) {
		if deps.HealthChecker == nil {
			c.String(http.StatusOK, "ok")
			return
		}
		result := deps.HealthChecker.LivenessCheck()
		c.JSON(http.StatusOK, result)
	})

	// @Summary Readiness 检查
	// @Description 服务就绪检查，检查依赖服务（PostgreSQL、Redis）状态
	// @Tags Health
	// @Produce json
	// @Success 200 {object} dto.HealthResponse
	// @Failure 503 {object} dto.HealthResponse
	// @Router /readyz [get]
	r.GET("/readyz", func(c *gin.Context) {
		if deps.HealthChecker == nil {
			c.String(http.StatusOK, "ok")
			return
		}
		result := deps.HealthChecker.ReadinessCheck(c.Request.Context())
		if result.Status == "error" {
			c.JSON(http.StatusServiceUnavailable, result)
			return
		}
		c.JSON(http.StatusOK, result)
	})

	// Prometheus metrics 端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API 路由（需要在静态文件服务之前注册，确保优先匹配）
	api := r.Group("/api/v1")
	{
		// -------- workers 配置和注册 --------
		// @Summary 获取 Worker 列表
		// @Description 获取所有已注册的 Worker 配置列表
		// @Tags Workers
		// @Produce json
		// @Success 200 {object} dto.WorkerListResponse
		// @Router /workers [get]
		api.GET("/workers", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"items": deps.WorkerStore.List()})
		})

		// 获取 worker 统计信息
		// @Summary 获取 Worker 统计信息
		// @Description 获取指定 Worker 的任务执行统计信息
		// @Tags Workers
		// @Produce json
		// @Param worker_name path string true "Worker 名称"
		// @Success 200 {object} dto.WorkerStatsResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Failure 501 {object} dto.ErrorResponse
		// @Router /workers/{worker_name}/stats [get]
		api.GET("/workers/:worker_name/stats", middleware.ValidateWorkerNameParam(), func(c *gin.Context) {
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			workerName := c.Param("worker_name")
			stats, err := deps.TaskRepo.GetWorkerStats(c.Request.Context(), workerName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"stats": stats})
		})

		// 获取 worker 时间序列统计
		// @Summary 获取 Worker 时间序列统计
		// @Description 获取指定 Worker 的时间序列统计数据
		// @Tags Workers
		// @Produce json
		// @Param worker_name path string true "Worker 名称"
		// @Param hours query int false "统计小时数" default(24)
		// @Success 200 {object} dto.WorkerTimeSeriesResponse
		// @Failure 501 {object} dto.ErrorResponse
		// @Router /workers/{worker_name}/timeseries [get]
		api.GET("/workers/:worker_name/timeseries", middleware.ValidateWorkerNameParam(), func(c *gin.Context) {
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			workerName := c.Param("worker_name")
			hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

			timeseries, err := deps.TaskRepo.GetWorkerTimeSeriesStats(c.Request.Context(), workerName, hours)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"timeseries": timeseries})
		})

		// Upsert worker 配置
		// @Summary 创建或更新 Worker
		// @Description 创建新的 Worker 或更新已存在的 Worker 配置
		// @Tags Workers
		// @Accept json
		// @Produce json
		// @Param request body dto.CreateWorkerRequest true "Worker 配置"
		// @Success 200 {object} dto.WorkerResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Router /workers [post]
		api.POST("/workers", func(c *gin.Context) {
			var req workers.Config
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			item, err := deps.WorkerStore.Upsert(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if deps.WorkerRepo != nil {
				if err := deps.WorkerRepo.Upsert(c.Request.Context(), item); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{"item": item})
		})

		// 获取单个 worker 配置
		// @Summary 获取 Worker 详情
		// @Description 根据 worker_name 获取 Worker 配置详情
		// @Tags Workers
		// @Produce json
		// @Param worker_name path string true "Worker 名称"
		// @Success 200 {object} dto.WorkerResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Router /workers/{worker_name} [get]
		api.GET("/workers/:worker_name", middleware.ValidateWorkerNameParam(), func(c *gin.Context) {
			workerName := c.Param("worker_name")
			item, ok := deps.WorkerStore.Get(workerName)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "worker 不存在"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"item": item})
		})

		// Worker 心跳更新
		// @Summary 更新 Worker 心跳
		// @Description 更新指定 Worker 的心跳时间
		// @Tags Workers
		// @Produce json
		// @Param worker_name path string true "Worker 名称"
		// @Success 200 {object} dto.HeartbeatResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Router /workers/{worker_name}/heartbeat [post]
		api.POST("/workers/:worker_name/heartbeat", middleware.ValidateWorkerNameParam(), func(c *gin.Context) {
			workerName := c.Param("worker_name")

			// 检查 worker 是否存在
			config, ok := deps.WorkerStore.Get(workerName)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "worker 不存在"})
				return
			}

			// 更新心跳时间
			now := time.Now()
			config.LastHeartbeatAt = &now

			// 更新内存存储
			_, err := deps.WorkerStore.Upsert(config)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 更新数据库
			if deps.WorkerRepo != nil {
				if err := deps.WorkerRepo.UpdateHeartbeat(c.Request.Context(), workerName, now); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"status":       "ok",
				"worker_name":  workerName,
				"heartbeat_at": now,
			})
		})

		// SDK worker 注册接口
		// @Summary 注册 Worker
		// @Description Worker SDK 自动注册接口
		// @Tags Workers
		// @Accept json
		// @Produce json
		// @Param request body dto.RegisterWorkerRequest true "Worker 注册信息"
		// @Success 200 {object} dto.SuccessResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Router /workers/register [post]
		api.POST("/workers/register", func(c *gin.Context) {
			var req struct {
				WorkerName        string         `json:"worker_name" binding:"required"`
				BaseURL           string         `json:"base_url"`
				RedisAddr         string         `json:"redis_addr"`
				Concurrency       int32          `json:"concurrency"`
				Queues            map[string]int `json:"queues" binding:"required"`
				DefaultRetryCount int32          `json:"default_retry_count"`
				DefaultTimeout    int32          `json:"default_timeout"`
				DefaultDelay      int32          `json:"default_delay"`
				Overwrite         bool           `json:"overwrite"` // true 时覆盖已有配置（谨慎使用）
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 检查是否已存在
			_, exists := deps.WorkerStore.Get(req.WorkerName)
			if exists && !req.Overwrite {
				c.JSON(http.StatusOK, gin.H{
					"status":  "ok",
					"message": "worker 已存在，跳过注册（使用 overwrite=true 强制更新）",
				})
				return
			}

			now := time.Now()
			config := workers.Config{
				WorkerName:        req.WorkerName,
				BaseURL:           req.BaseURL,
				RedisAddr:         req.RedisAddr,
				Concurrency:       req.Concurrency,
				Queues:            req.Queues,
				DefaultRetryCount: req.DefaultRetryCount,
				DefaultTimeout:    req.DefaultTimeout,
				DefaultDelay:      req.DefaultDelay,
				IsEnabled:         true,
				LastHeartbeatAt:   &now,
			}

			item, err := deps.WorkerStore.Upsert(config)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if deps.WorkerRepo != nil {
				if err := deps.WorkerRepo.Upsert(c.Request.Context(), item); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"worker": item,
			})
		})

		// -------- tasks（入队/查询/重放）--------

		// 创建任务
		// @Summary 创建任务
		// @Description 创建新的异步任务并入队到 Asynq
		// @Tags Tasks
		// @Accept json
		// @Produce json
		// @Param request body dto.CreateTaskRequest true "任务创建请求"
		// @Success 200 {object} dto.CreateTaskResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 503 {object} dto.ErrorResponse
		// @Router /tasks [post]
		api.POST("/tasks", func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}

			var req struct {
				WorkerName   string          `json:"worker_name" binding:"required"`
				Queue        string          `json:"queue" binding:"required"`
				TaskID       string          `json:"task_id"` // 可选，默认生成
				Payload      json.RawMessage `json:"payload" binding:"required"`
				DelaySeconds int32           `json:"delay_seconds"`
				RunAt        *time.Time      `json:"run_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 验证 worker_name 格式
			if !middleware.ValidateWorkerName(req.WorkerName) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "worker_name 格式无效"})
				return
			}

			// 验证 queue 格式
			if !middleware.ValidateQueueName(req.Queue) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "queue 格式无效"})
				return
			}

			// 验证 task_id 格式（如果提供）
			if req.TaskID != "" && !middleware.ValidateTaskID(req.TaskID) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "task_id 格式无效"})
				return
			}

			// 验证 payload 大小
			if len(req.Payload) > middleware.MaxPayloadSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": "payload 过大，最大 2MB"})
				return
			}

			// 验证 worker 是否存在
			workerCfg, ok := deps.WorkerStore.Get(req.WorkerName)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "worker 不存在"})
				return
			}
			if !workerCfg.IsEnabled {
				c.JSON(http.StatusBadRequest, gin.H{"error": "worker 未启用"})
				return
			}

			// 验证 queue 是否在 worker 的配置中
			if !deps.WorkerStore.HasQueue(req.WorkerName, req.Queue) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "队列不存在于 worker 配置中"})
				return
			}

			// 生成或使用提供的 task_id
			taskID := req.TaskID
			if taskID == "" {
				taskID = asynqx.NewTaskID()
			}

			// 构造完整队列名：workerName:queueName
			fullQueue := req.WorkerName + ":" + req.Queue

			// 构造 task payload
			var msg = struct {
				ID      string          `json:"id,omitempty"`
				TaskID  string          `json:"task_id,omitempty"`
				Payload json.RawMessage `json:"payload"`
			}{
				ID:      taskID,
				TaskID:  taskID,
				Payload: req.Payload,
			}
			b, _ := json.Marshal(msg)

			// 入队到 asynq
			t := asynq.NewTask(fullQueue, b)
			p := asynqx.EnqueueParams{
				TaskType:       fullQueue,
				TaskKey:        taskID,
				Queue:          fullQueue,
				MaxRetry:       workerCfg.DefaultRetryCount,
				TimeoutSeconds: workerCfg.DefaultTimeout,
				DelaySeconds:   req.DelaySeconds,
				Payload:        req.Payload,
			}
			if req.RunAt != nil {
				p.RunAt = *req.RunAt
			}

			info, err := deps.AsynqClient.Enqueue(t, asynqx.EnqueueOptions(p)...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 记录到数据库
			if deps.TaskRepo != nil {
				_ = deps.TaskRepo.UpsertTask(c.Request.Context(), repository.Task{
					TaskID:      taskID,
					WorkerName:  req.WorkerName,
					Queue:       fullQueue,
					Priority:    0, // 不再使用 priority
					Payload:     req.Payload,
					Status:      string(model.TaskStatusPending),
					LastAttempt: 0,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"task_id":       taskID,
				"worker_name":   req.WorkerName,
				"queue":         req.Queue,
				"full_queue":    fullQueue,
				"asynq_task_id": info.ID,
				"status":        "enqueued",
			})
		})

		// 任务列表
		// @Summary 查询任务列表
		// @Description 分页查询任务列表，支持多条件过滤
		// @Tags Tasks
		// @Produce json
		// @Param worker_name query string false "Worker 名称"
		// @Param status query string false "任务状态"
		// @Param queue query string false "队列名称"
		// @Param limit query int false "每页数量" default(50)
		// @Param offset query int false "偏移量" default(0)
		// @Success 200 {object} dto.TaskListResponse
		// @Failure 501 {object} dto.ErrorResponse
		// @Router /tasks [get]
		api.GET("/tasks", func(c *gin.Context) {
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			items, err := deps.TaskRepo.ListTasks(c.Request.Context(), repository.ListTasksFilter{
				WorkerName: c.DefaultQuery("worker_name", ""),
				Status:     c.DefaultQuery("status", ""),
				Queue:      c.DefaultQuery("queue", ""),
				Limit:      limit,
				Offset:     offset,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			total, err := deps.TaskRepo.CountTasks(c.Request.Context(), repository.ListTasksFilter{
				WorkerName: c.DefaultQuery("worker_name", ""),
				Status:     c.DefaultQuery("status", ""),
				Queue:      c.DefaultQuery("queue", ""),
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
		})

		// 任务详情
		// @Summary 获取任务详情
		// @Description 根据 task_id 获取任务详细信息及执行历史
		// @Tags Tasks
		// @Produce json
		// @Param task_id path string true "任务 ID"
		// @Success 200 {object} dto.TaskResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Failure 501 {object} dto.ErrorResponse
		// @Router /tasks/{task_id} [get]
		api.GET("/tasks/:task_id", middleware.ValidateTaskIDParam(), func(c *gin.Context) {
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			taskID := c.Param("task_id")
			t, err := deps.TaskRepo.GetTask(c.Request.Context(), taskID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "task 不存在"})
				return
			}
			attempts, _ := deps.TaskRepo.ListAttempts(c.Request.Context(), taskID, 50)
			c.JSON(http.StatusOK, gin.H{"item": t, "attempts": attempts})
		})

		// 重放任务
		// @Summary 重放任务
		// @Description 将已完成或失败的任务重新入队执行
		// @Tags Tasks
		// @Accept json
		// @Produce json
		// @Param task_id path string true "任务 ID"
		// @Param request body dto.ReplayTaskRequest false "重放参数"
		// @Success 200 {object} dto.ReplayTaskResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Failure 503 {object} dto.ErrorResponse
		// @Router /tasks/{task_id}/replay [post]
		api.POST("/tasks/:task_id/replay", middleware.ValidateTaskIDParam(), func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			taskID := c.Param("task_id")
			t, err := deps.TaskRepo.GetTask(c.Request.Context(), taskID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "task 不存在"})
				return
			}

			// 验证 worker 配置
			workerCfg, ok := deps.WorkerStore.Get(t.WorkerName)
			if !ok || !workerCfg.IsEnabled {
				c.JSON(http.StatusBadRequest, gin.H{"error": "worker 不存在或未启用"})
				return
			}

			// 重放语义：创建一个新的 task_id
			newTaskID := asynqx.NewTaskID()
			var msg = struct {
				ID      string          `json:"id,omitempty"`
				TaskID  string          `json:"task_id,omitempty"`
				Payload json.RawMessage `json:"payload"`
			}{
				ID:      newTaskID,
				TaskID:  newTaskID,
				Payload: t.Payload,
			}
			b, _ := json.Marshal(msg)

			info, err := deps.AsynqClient.Enqueue(asynq.NewTask(t.Queue, b), asynqx.EnqueueOptions(asynqx.EnqueueParams{
				TaskType:       t.Queue,
				TaskKey:        newTaskID,
				Queue:          t.Queue,
				MaxRetry:       workerCfg.DefaultRetryCount,
				TimeoutSeconds: workerCfg.DefaultTimeout,
				Payload:        t.Payload,
			})...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			_ = deps.TaskRepo.UpsertTask(c.Request.Context(), repository.Task{
				TaskID:      newTaskID,
				WorkerName:  t.WorkerName,
				Queue:       t.Queue,
				Priority:    t.Priority,
				Payload:     t.Payload,
				Status:      "pending",
				LastAttempt: 0,
				LastError:   "",
			})

			c.JSON(http.StatusOK, gin.H{
				"task_id":       newTaskID,
				"asynq_task_id": info.ID,
				"status":        "replayed",
			})
		})

		// Worker 上报任务执行结果
		// @Summary 上报任务执行状态
		// @Description Worker 上报任务执行的详细状态（running、success、fail）
		// @Tags Tasks
		// @Accept json
		// @Produce json
		// @Param task_id path string true "任务 ID"
		// @Param request body dto.ReportAttemptRequest true "执行状态"
		// @Success 200 {object} dto.SuccessResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Router /tasks/{task_id}/report-attempt [post]
		api.POST("/tasks/:task_id/report-attempt", middleware.ValidateTaskIDParam(), func(c *gin.Context) {
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			taskID := c.Param("task_id")
			var req struct {
				Attempt     int        `json:"attempt" binding:"required"`
				Status      string     `json:"status" binding:"required"` // running/success/fail/dead
				AsynqTaskID string     `json:"asynq_task_id"`
				Error       string     `json:"error"`
				WorkerName  string     `json:"worker_name"`
				StartedAt   *time.Time `json:"started_at"`
				FinishedAt  *time.Time `json:"finished_at"`
				DurationMs  *int       `json:"duration_ms"`
				TraceID     string     `json:"trace_id"`
				SpanID      string     `json:"span_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			status := model.TaskStatus(req.Status)
			if !status.Valid() || status == model.TaskStatusPending {
				c.JSON(http.StatusBadRequest, gin.H{"error": "status 非法（允许：running/success/fail/dead）"})
				return
			}

			started := time.Now()
			if req.StartedAt != nil {
				started = *req.StartedAt
			}

			if err := deps.TaskRepo.InsertAttempt(c.Request.Context(), repository.Attempt{
				TaskID:      taskID,
				AsynqTaskID: req.AsynqTaskID,
				Attempt:     req.Attempt,
				Status:      req.Status,
				StartedAt:   started,
				FinishedAt:  req.FinishedAt,
				DurationMs:  req.DurationMs,
				Error:       req.Error,
				WorkerName:  req.WorkerName,
				TraceID:     req.TraceID,
				SpanID:      req.SpanID,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			_ = deps.TaskRepo.UpdateTaskStatus(c.Request.Context(), taskID, string(status), req.Attempt, req.Error, req.WorkerName)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// -------- 批量任务管理 --------

		// 批量重试失败任务
		// @Summary 批量重试失败任务
		// @Description 批量重试指定条件的失败任务
		// @Tags Tasks
		// @Accept json
		// @Produce json
		// @Param request body dto.BatchRetryRequest true "重试条件"
		// @Success 200 {object} dto.BatchRetryResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 501 {object} dto.ErrorResponse
		// @Router /tasks/batch-retry [post]
		api.POST("/tasks/batch-retry", func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}
			if deps.TaskRepo == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "Postgres 未配置"})
				return
			}

			var req struct {
				WorkerName string   `json:"worker_name"`
				Status     string   `json:"status"`   // 默认 "fail"
				TaskIDs    []string `json:"task_ids"` // 可选，指定任务ID列表
				Limit      int      `json:"limit"`    // 最多重试多少个任务，默认100
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 默认值
			if req.Status == "" {
				req.Status = "fail"
			}
			if req.Limit <= 0 || req.Limit > 1000 {
				req.Limit = 100
			}

			var tasks []repository.Task
			var err error

			// 根据条件查询失败任务
			if len(req.TaskIDs) > 0 {
				// 根据任务ID列表查询
				for _, taskID := range req.TaskIDs {
					t, e := deps.TaskRepo.GetTask(c.Request.Context(), taskID)
					if e == nil {
						tasks = append(tasks, *t)
					}
				}
			} else {
				// 查询失败任务列表
				tasks, err = deps.TaskRepo.ListTasks(c.Request.Context(), repository.ListTasksFilter{
					WorkerName: req.WorkerName,
					Status:     req.Status,
					Limit:      req.Limit,
					Offset:     0,
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			// 批量重新入队
			var successCount int
			var failedTasks []string

			for _, task := range tasks {
				// 验证 worker 配置
				workerCfg, ok := deps.WorkerStore.Get(task.WorkerName)
				if !ok || !workerCfg.IsEnabled {
					failedTasks = append(failedTasks, task.TaskID)
					continue
				}

				// 创建新任务ID用于重试
				newTaskID := asynqx.NewTaskID()
				var msg = struct {
					ID      string          `json:"id,omitempty"`
					TaskID  string          `json:"task_id,omitempty"`
					Payload json.RawMessage `json:"payload"`
				}{
					ID:      newTaskID,
					TaskID:  newTaskID,
					Payload: task.Payload,
				}
				b, _ := json.Marshal(msg)

				// 入队
				_, err := deps.AsynqClient.Enqueue(asynq.NewTask(task.Queue, b), asynqx.EnqueueOptions(asynqx.EnqueueParams{
					TaskType:       task.Queue,
					TaskKey:        newTaskID,
					Queue:          task.Queue,
					MaxRetry:       workerCfg.DefaultRetryCount,
					TimeoutSeconds: workerCfg.DefaultTimeout,
					Payload:        task.Payload,
				})...)

				if err != nil {
					failedTasks = append(failedTasks, task.TaskID)
					continue
				}

				// 记录新任务到数据库
				_ = deps.TaskRepo.UpsertTask(c.Request.Context(), repository.Task{
					TaskID:      newTaskID,
					WorkerName:  task.WorkerName,
					Queue:       task.Queue,
					Priority:    task.Priority,
					Payload:     task.Payload,
					Status:      "pending",
					LastAttempt: 0,
				})

				successCount++
			}

			c.JSON(http.StatusOK, gin.H{
				"status":        "ok",
				"total":         len(tasks),
				"success_count": successCount,
				"failed_count":  len(failedTasks),
				"failed_tasks":  failedTasks,
			})
		})

		// -------- Asynq 队列管理 --------

		// 查询队列状态
		// @Summary 查询队列状态
		// @Description 获取指定 Worker 的所有队列状态统计
		// @Tags Queues
		// @Produce json
		// @Param worker_name query string true "Worker 名称"
		// @Success 200 {object} dto.QueueStatsResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 404 {object} dto.ErrorResponse
		// @Router /queues/stats [get]
		api.GET("/queues/stats", func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}

			workerName := c.Query("worker_name")
			if workerName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "worker_name 参数必填"})
				return
			}

			// 获取 worker 配置
			workerCfg, ok := deps.WorkerStore.Get(workerName)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "worker 不存在"})
				return
			}

			// 创建 Inspector
			inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})
			defer inspector.Close()

			queuesStats := make([]gin.H, 0)
			for queueName := range workerCfg.Queues {
				fullQueue := workerName + ":" + queueName

				// 获取队列信息
				queueInfo, err := inspector.GetQueueInfo(fullQueue)
				if err != nil {
					continue
				}

				queuesStats = append(queuesStats, gin.H{
					"queue":        queueName,
					"full_queue":   fullQueue,
					"pending":      queueInfo.Pending,
					"active":       queueInfo.Active,
					"scheduled":    queueInfo.Scheduled,
					"retry":        queueInfo.Retry,
					"archived":     queueInfo.Archived,
					"completed":    queueInfo.Completed,
					"aggregating":  queueInfo.Aggregating,
					"processed":    queueInfo.Processed,
					"failed":       queueInfo.Failed,
					"paused":       queueInfo.Paused,
					"latency_msec": queueInfo.Latency.Milliseconds(),
					"memory_usage": queueInfo.MemoryUsage,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"worker_name": workerName,
				"queues":      queuesStats,
			})
		})

		// 清空队列（删除所有 pending 任务）
		// @Summary 清空队列
		// @Description 清空指定 Worker 的队列（删除所有待处理任务）
		// @Tags Queues
		// @Accept json
		// @Produce json
		// @Param request body dto.ClearQueueRequest true "清空队列请求"
		// @Success 200 {object} dto.ClearQueueResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 503 {object} dto.ErrorResponse
		// @Router /queues/clear [post]
		api.POST("/queues/clear", func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}

			var req struct {
				WorkerName string `json:"worker_name" binding:"required"`
				QueueName  string `json:"queue_name"` // 可选，不指定则清空所有队列
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 获取 worker 配置
			workerCfg, ok := deps.WorkerStore.Get(req.WorkerName)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "worker 不存在"})
				return
			}

			// 创建 Inspector
			inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})
			defer inspector.Close()

			var clearedQueues []string
			var totalDeleted int

			// 确定要清空的队列
			queuesToClear := make(map[string]bool)
			if req.QueueName != "" {
				// 验证队列是否存在
				if _, exists := workerCfg.Queues[req.QueueName]; !exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "队列不存在于 worker 配置中"})
					return
				}
				queuesToClear[req.QueueName] = true
			} else {
				// 清空所有队列
				for qName := range workerCfg.Queues {
					queuesToClear[qName] = true
				}
			}

			// 删除队列中的任务
			for qName := range queuesToClear {
				fullQueue := req.WorkerName + ":" + qName

				// 删除 pending 任务
				deleted, err := inspector.DeleteAllPendingTasks(fullQueue)
				if err != nil {
					continue
				}

				clearedQueues = append(clearedQueues, qName)
				totalDeleted += deleted
			}

			c.JSON(http.StatusOK, gin.H{
				"status":         "ok",
				"worker_name":    req.WorkerName,
				"cleared_queues": clearedQueues,
				"total_deleted":  totalDeleted,
			})
		})

		// 清空失败任务队列（archived/dead letter queue）
		// @Summary 清空死信队列
		// @Description 清空指定 Worker 的死信队列（已达最大重试次数的失败任务）
		// @Tags Queues
		// @Accept json
		// @Produce json
		// @Param request body dto.ClearDeadQueueRequest true "清空死信队列请求"
		// @Success 200 {object} dto.ClearDeadQueueResponse
		// @Failure 400 {object} dto.ErrorResponse
		// @Failure 503 {object} dto.ErrorResponse
		// @Router /queues/clear-dead [post]
		api.POST("/queues/clear-dead", func(c *gin.Context) {
			if deps.AsynqClient == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "asynq client 未配置"})
				return
			}

			var req struct {
				WorkerName string `json:"worker_name" binding:"required"`
				QueueName  string `json:"queue_name"` // 可选
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 获取 worker 配置
			workerCfg, ok := deps.WorkerStore.Get(req.WorkerName)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "worker 不存在"})
				return
			}

			// 创建 Inspector
			inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})
			defer inspector.Close()

			var clearedQueues []string
			var totalDeleted int

			// 确定要清空的队列
			queuesToClear := make(map[string]bool)
			if req.QueueName != "" {
				if _, exists := workerCfg.Queues[req.QueueName]; !exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "队列不存在于 worker 配置中"})
					return
				}
				queuesToClear[req.QueueName] = true
			} else {
				for qName := range workerCfg.Queues {
					queuesToClear[qName] = true
				}
			}

			// 删除 archived 任务
			for qName := range queuesToClear {
				fullQueue := req.WorkerName + ":" + qName

				// 删除 archived (dead letter) 任务
				deleted, err := inspector.DeleteAllArchivedTasks(fullQueue)
				if err != nil {
					continue
				}

				clearedQueues = append(clearedQueues, qName)
				totalDeleted += deleted
			}

			c.JSON(http.StatusOK, gin.H{
				"status":         "ok",
				"worker_name":    req.WorkerName,
				"cleared_queues": clearedQueues,
				"total_deleted":  totalDeleted,
			})
		})
	}

	// Web UI 静态文件服务（放在最后，作为默认路由）
	if deps.WebFS != nil {
		// 获取嵌入的文件系统，去掉 "webui" 前缀
		distFS, err := fs.Sub(*deps.WebFS, "webui")
		if err != nil {
			log.Printf("警告: Web UI 加载失败: %v", err)
		} else {
			// 创建文件服务器
			fileServer := http.FileServer(http.FS(distFS))

			// 使用 NoRoute 来处理所有未匹配的路由，提供 SPA 支持
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path

				// 尝试打开文件
				f, err := distFS.Open(strings.TrimPrefix(path, "/"))
				if err == nil {
					// 文件存在，关闭并使用文件服务器
					f.Close()
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}

				// 文件不存在，返回 index.html（SPA 路由支持）
				c.Request.URL.Path = "/"
				fileServer.ServeHTTP(c.Writer, c.Request)
			})

			log.Printf("✅ Web UI 已挂载到根路径 /")
		}
	}

	return r
}
