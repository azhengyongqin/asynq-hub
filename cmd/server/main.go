package main

import (
	"context"
	"embed"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	_ "github.com/azhengyongqin/asynq-hub/docs" // Swagger docs
	"github.com/azhengyongqin/asynq-hub/internal/config"
	"github.com/azhengyongqin/asynq-hub/internal/healthcheck"
	"github.com/azhengyongqin/asynq-hub/internal/logger"
	asynqx "github.com/azhengyongqin/asynq-hub/internal/queue"
	"github.com/azhengyongqin/asynq-hub/internal/repository"
	httpserver "github.com/azhengyongqin/asynq-hub/internal/server"
	"github.com/azhengyongqin/asynq-hub/internal/storage/postgres"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// WebFS 嵌入 web/dist 目录下的所有静态文件
//
//go:embed webui
var WebFS embed.FS

// @title Asynq-Hub API
// @version 1.0.0
// @description 分布式任务管理系统 - 基于 Asynq 和 PostgreSQL 的任务调度平台
// @contact.name Asynq-Hub Support
// @license.name MIT
// @BasePath /api/v1
// @schemes http https
// @host localhost:28080

// 说明：
// - MVP 先用一个进程启动 Gin(HTTP)，便于本地与容器部署。

func main() {
	// 初始化结构化日志（开发模式）
	if err := logger.Init(false); err != nil {
		logger.L.Fatal().Err(err).Msg("初始化日志失败")
		os.Exit(1)
	}
	defer logger.Sync()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.L.Fatal().Err(err).Msg("加载配置失败")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		logger.L.Fatal().Err(err).Msg("配置验证失败")
	}

	logger.L.Info().
		Str("http", cfg.HTTP.Addr).
		Str("grpc", cfg.GRPC.Addr).
		Msg("服务启动")

	httpAddr := cfg.HTTP.Addr
	redisAddr := cfg.Redis.Addr
	postgresDSN := cfg.Postgres.DSN

	// 确保 Redis 地址格式正确
	if !strings.HasPrefix(redisAddr, "redis://") && !strings.HasPrefix(redisAddr, "rediss://") {
		redisAddr = "redis://" + redisAddr + "/0"
	}

	// worker 配置：默认使用内存；若配置了 Postgres，则以 Postgres 作为持久化来源。
	workerStore := workers.NewStore()

	var (
		workerRepo *repository.WorkerRepo
		taskRepo   *repository.TaskRepo
	)

	// 使用配置的连接池参数
	dbCfg := postgres.DBConfig{
		MaxOpenConns:    int(cfg.DBPool.MaxConns),
		MaxIdleConns:    int(cfg.DBPool.MinConns),
		ConnMaxLifetime: cfg.DBPool.MaxConnLifetime,
		ConnMaxIdleTime: cfg.DBPool.MaxConnIdleTime,
	}

	db, err := postgres.NewDBWithConfig(context.Background(), postgresDSN, dbCfg)
	if err != nil {
		logger.L.Fatal().Err(err).Msg("连接数据库失败")
	}
	defer db.Close()

	workerRepo = repository.NewWorkerRepo(db.DB)
	taskRepo = repository.NewTaskRepo(db.DB)

	// 加载已注册的 workers
	cfgs, err := workerRepo.List(context.Background())
	if err != nil {
		logger.L.Fatal().Err(err).Msg("加载 worker 配置失败")
	}
	for _, c := range cfgs {
		// 转换队列组配置
		queueGroups := make([]workers.QueueGroupConfig, len(c.QueueGroups))
		for i, qg := range c.QueueGroups {
			queueGroups[i] = workers.QueueGroupConfig{
				Name:        qg.Name,
				Concurrency: qg.Concurrency,
				Priorities:  qg.Priorities,
			}
		}

		workerCfg := workers.Config{
			WorkerName:        c.WorkerName,
			BaseURL:           c.BaseURL,
			RedisAddr:         c.RedisAddr,
			QueueGroups:       queueGroups,
			DefaultRetryCount: c.DefaultRetryCount,
			DefaultTimeout:    c.DefaultTimeout,
			DefaultDelay:      c.DefaultDelay,
			IsEnabled:         c.IsEnabled,
			LastHeartbeatAt:   c.LastHeartbeatAt,
		}
		_, _ = workerStore.Upsert(workerCfg)
	}
	logger.L.Info().Int("count", len(cfgs)).Msg("已加载 worker 配置")

	// Asynq client：用于 HTTP 入队
	redisOpt, err := asynqx.NewRedisConnOpt(redisAddr)
	if err != nil {
		logger.L.Fatal().Err(err).Msg("解析 Redis URI 失败")
	}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	// 创建健康检查器
	healthChecker := healthcheck.NewHealthChecker(db.DB, asynqClient, redisAddr)

	httpSrv := &http.Server{
		Addr: httpAddr,
		Handler: httpserver.NewRouter(httpserver.Deps{
			WorkerStore:   workerStore,
			AsynqClient:   asynqClient,
			WorkerRepo:    workerRepo,
			TaskRepo:      taskRepo,
			HealthChecker: healthChecker,
			WebFS:         &WebFS,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.L.Info().Str("addr", httpAddr).Msg("HTTP 服务监听")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal().Err(err).Msg("HTTP 服务错误")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	logger.L.Info().Msg("服务已优雅关闭")
}
