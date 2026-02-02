package httpserver

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/azhengyongqin/asynq-hub/internal/healthcheck"
	"github.com/azhengyongqin/asynq-hub/internal/middleware"
	"github.com/azhengyongqin/asynq-hub/internal/repository"
	"github.com/azhengyongqin/asynq-hub/internal/server/handler"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

type Deps struct {
	WorkerStore *workers.Store

	// AsynqClient 用于入队
	AsynqClient *asynq.Client

	// 可选：若提供则会把数据落到 Postgres（用于列表/详情/报表）
	WorkerRepo repository.WorkerRepository
	TaskRepo   repository.TaskRepository

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

	// 创建各个 handler 实例
	healthHandler := handler.NewHealthHandler(deps.HealthChecker)
	workerHandler := handler.NewWorkerHandler(deps.WorkerStore, deps.WorkerRepo, deps.TaskRepo)
	taskHandler := handler.NewTaskHandler(deps.AsynqClient, deps.TaskRepo, deps.WorkerStore)
	queueHandler := handler.NewQueueHandler(deps.AsynqClient, deps.WorkerStore)

	// 健康检查路由
	r.GET("/healthz", healthHandler.Liveness)
	r.GET("/readyz", healthHandler.Readiness)

	// Prometheus metrics 端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API 路由（需要在静态文件服务之前注册，确保优先匹配）
	api := r.Group("/api/v1")
	{
		// Worker 相关路由
		api.GET("/workers", workerHandler.ListWorkers)
		api.GET("/workers/:worker_name", middleware.ValidateWorkerNameParam(), workerHandler.GetWorker)
		api.GET("/workers/:worker_name/stats", middleware.ValidateWorkerNameParam(), workerHandler.GetWorkerStats)
		api.GET("/workers/:worker_name/timeseries", middleware.ValidateWorkerNameParam(), workerHandler.GetWorkerTimeSeries)
		api.POST("/workers", workerHandler.CreateOrUpdateWorker)
		api.POST("/workers/:worker_name/heartbeat", middleware.ValidateWorkerNameParam(), workerHandler.UpdateHeartbeat)
		api.POST("/workers/register", workerHandler.RegisterWorker)

		// Task 相关路由
		api.POST("/tasks", taskHandler.CreateTask)
		api.GET("/tasks", taskHandler.ListTasks)
		api.GET("/tasks/:task_id", middleware.ValidateTaskIDParam(), taskHandler.GetTask)
		api.POST("/tasks/:task_id/replay", middleware.ValidateTaskIDParam(), taskHandler.ReplayTask)
		api.POST("/tasks/:task_id/report-attempt", middleware.ValidateTaskIDParam(), taskHandler.ReportAttempt)
		api.POST("/tasks/batch-retry", taskHandler.BatchRetry)

		// Queue 相关路由
		api.GET("/queues/stats", queueHandler.GetQueueStats)
		api.POST("/queues/clear", queueHandler.ClearQueue)
		api.POST("/queues/clear-dead", queueHandler.ClearDeadQueue)
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
