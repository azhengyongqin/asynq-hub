package main

import (
	"context"
	"embed"
	"log"
	"net/http"
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
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.S.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		logger.S.Fatalf("配置验证失败: %v", err)
	}

	logger.S.Infof("服务启动，HTTP: %s, gRPC: %s", cfg.HTTP.Addr, cfg.GRPC.Addr)

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
	poolCfg := postgres.PoolConfig{
		MaxConns:          cfg.DBPool.MaxConns,
		MinConns:          cfg.DBPool.MinConns,
		MaxConnLifetime:   cfg.DBPool.MaxConnLifetime,
		MaxConnIdleTime:   cfg.DBPool.MaxConnIdleTime,
		HealthCheckPeriod: cfg.DBPool.HealthCheckPeriod,
	}

	pgPool, err := postgres.NewPoolWithConfig(context.Background(), postgresDSN, poolCfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pgPool.Close()

	workerRepo = repository.NewWorkerRepo(pgPool.Pool)
	taskRepo = repository.NewTaskRepo(pgPool.Pool)

	// 加载已注册的 workers
	cfgs, err := workerRepo.List(context.Background())
	if err != nil {
		log.Fatalf("load workers: %v", err)
	}
	for _, c := range cfgs {
		_, _ = workerStore.Upsert(c)
	}
	logger.S.Infof("已加载 %d 个 worker 配置", len(cfgs))

	// Asynq client：用于 HTTP 入队
	redisOpt, err := asynqx.NewRedisConnOpt(redisAddr)
	if err != nil {
		log.Fatalf("parse redis uri: %v", err)
	}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	// 创建健康检查器
	healthChecker := healthcheck.NewHealthChecker(pgPool.Pool, asynqClient, redisAddr)

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
		logger.S.Infof("HTTP 服务监听: %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.S.Fatalf("HTTP 服务错误: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	logger.S.Info("服务已优雅关闭")
}
