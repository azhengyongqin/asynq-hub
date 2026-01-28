package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

var (
	envLoaded bool
	envLoadMu sync.Mutex
)

func init() {
	// 在包初始化时尝试加载 .env 文件
	_ = loadEnvFile()
}

// loadEnvFile 尝试从项目根目录加载 .env 文件
// 会尝试多个可能的路径，找到第一个存在的 .env 文件
func loadEnvFile() error {
	envLoadMu.Lock()
	defer envLoadMu.Unlock()

	if envLoaded {
		return nil
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 尝试多个可能的 .env 文件路径（按优先级）
	possiblePaths := []string{
		// 1. 当前目录
		filepath.Join(wd, ".env"),
		// 2. 从 go-worker-sdk 目录向上查找项目根目录
		filepath.Join(wd, "..", ".env"),
		// 3. 从 go-worker-sdk/asynqhub-worker 向上查找项目根目录
		filepath.Join(wd, "..", "..", ".env"),
		// 4. 从 go-worker-sdk/asynqhub-client 向上查找项目根目录
		filepath.Join(wd, "..", "..", "..", ".env"),
	}

	// 查找第一个存在的 .env 文件
	var envPath string
	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			envPath = absPath
			break
		}
	}

	// 如果没有找到 .env 文件，返回 nil（允许通过环境变量配置）
	if envPath == "" {
		envLoaded = true
		return nil
	}

	// 加载 .env 文件
	if err := godotenv.Load(envPath); err != nil {
		return err
	}

	log.Printf("[asynqhub-worker] 已加载环境变量文件: %s", envPath)
	envLoaded = true
	return nil
}

// Worker：对齐你给的示例 main.go
//
// 目标：
// - 用户只需要配置 baseURL、taskType
// - 用户只需要写 HandleFunc(taskType, func(ctx, *asynq.Task) error)
// - SDK 自动上报 attempt/状态，且可选注册 task_type（默认 create-only）
type Worker struct {
	workerName string

	baseURL  string
	redisURI string
	redisOpt asynq.RedisClientOpt

	concurrency int
	queues      map[string]int

	defaultRetryCount int
	defaultTimeout    time.Duration
	defaultDelay      time.Duration

	autoRegister bool
	overwriteReg bool

	mux    *asynq.ServeMux
	server *asynq.Server
	client *asynq.Client

	reporter  Reporter
	registrar Registrar

	registered     []WorkerConfig
	registeredOnce bool
}

type Option func(*Worker)

func WithBaseURL(baseURL string) Option { return func(w *Worker) { w.baseURL = baseURL } }
func WithRedisAddr(redis string) Option {
	return func(w *Worker) {
		w.redisURI = redis
	}
}
func WithConcurrency(n int) Option              { return func(w *Worker) { w.concurrency = n } }
func WithQueues(q map[string]int) Option        { return func(w *Worker) { w.queues = q } }
func WithDefaultRetryCount(n int) Option        { return func(w *Worker) { w.defaultRetryCount = n } }
func WithDefaultTimeout(d time.Duration) Option { return func(w *Worker) { w.defaultTimeout = d } }
func WithDefaultDelay(d time.Duration) Option   { return func(w *Worker) { w.defaultDelay = d } }
func WithoutAutoRegister() Option               { return func(w *Worker) { w.autoRegister = false } }
func WithRegisterOverwrite() Option             { return func(w *Worker) { w.overwriteReg = true } }

// New：与示例一致
// 第一个参数 workerName（用于日志与上报）
func New(workerName string, opts ...Option) (*Worker, error) {
	w := &Worker{
		workerName:        workerName,
		redisURI:          os.Getenv("REDIS_ADDR"),
		concurrency:       20,
		queues:            map[string]int{},
		defaultRetryCount: 3,
		defaultTimeout:    30 * time.Second,
		defaultDelay:      0,
		autoRegister:      true,
		overwriteReg:      false,
		mux:               asynq.NewServeMux(),
	}
	for _, o := range opts {
		o(w)
	}

	// 强制使用 URI：redis://host:port/db
	// 默认使用 docker-compose.yml 中的 redis 配置（端口 16379）
	if w.redisURI == "" {
		w.redisURI = "redis://localhost:16379/0"
	}
	connOpt, err := asynq.ParseRedisURI(w.redisURI)
	if err != nil {
		return nil, fmt.Errorf("parse redis uri: %w", err)
	}
	opt, ok := connOpt.(asynq.RedisClientOpt)
	if !ok {
		return nil, fmt.Errorf("unexpected redis conn opt type: %T", connOpt)
	}
	w.redisOpt = opt
	if w.concurrency <= 0 {
		w.concurrency = 20
	}
	if len(w.queues) == 0 {
		// 默认队列权重（如需按 task_type 分队列，用户可传 WithQueues）
		w.queues = map[string]int{"critical": 50, "default": 30, "crawler": 20}
	}

	if w.workerName == "" {
		w.workerName = DefaultWorkerName()
	}

	// 约定：asynq queue 使用 workerName 作为前缀，隔离不同 worker 的消费域。
	// 形如：workerName:web_crawl
	w.queues = w.prefixQueues(w.queues)

	w.reporter = Reporter{
		ControlPlaneURL: w.baseURL,
		WorkerName:      w.workerName,
	}
	w.registrar = Registrar{
		ControlPlaneURL: w.baseURL,
	}

	w.server = asynq.NewServer(
		w.redisOpt,
		asynq.Config{
			Concurrency: w.concurrency,
			Queues:      w.prefixQueues(w.queues),
		},
	)
	w.client = asynq.NewClient(w.redisOpt)

	return w, nil
}

func (w *Worker) queuePrefix() string { return w.workerName + ":" }

func (w *Worker) queueName(name string) string {
	if name == "" {
		return w.queuePrefix()
	}
	if strings.HasPrefix(name, w.queuePrefix()) {
		return name
	}
	return w.queuePrefix() + name
}

func (w *Worker) prefixQueues(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[w.queueName(k)] = v
	}
	return out
}

// HandleFunc：注册队列处理器
// SDK 会在外层自动上报 attempt/状态。
func (w *Worker) HandleFunc(queueName string, fn func(ctx context.Context, t *asynq.Task) error) {
	// 使用完整队列名（包含 workerName 前缀）
	fullQueueName := w.queueName(queueName)

	w.mux.HandleFunc(fullQueueName, func(ctx context.Context, t *asynq.Task) error {
		// 从 payload 尝试解析 task_id
		taskID := ""
		var task Task
		if err := json.Unmarshal(t.Payload(), &task); err == nil {
			taskID = task.TaskID
			if taskID == "" {
				taskID = task.ID
			}
		}

		// 没有 task_id 时使用 asynq task id
		asynqID, _ := asynq.GetTaskID(ctx)
		if taskID == "" {
			taskID = asynqID
		}

		retryCount, _ := asynq.GetRetryCount(ctx)
		attemptNo := retryCount + 1
		start := time.Now()

		_ = w.reporter.ReportAttempt(ctx, taskID, ReportAttemptRequest{
			Attempt:     attemptNo,
			Status:      string(TaskStatusRunning),
			AsynqTaskID: asynqID,
			WorkerName:  w.workerName,
			StartedAt:   &start,
		})

		err := fn(ctx, t)

		finished := time.Now()
		dur := int(finished.Sub(start).Milliseconds())
		status := string(TaskStatusSuccess)
		errMsg := ""
		if err != nil {
			status = string(TaskStatusFail)
			errMsg = err.Error()
		}
		_ = w.reporter.ReportAttempt(ctx, taskID, ReportAttemptRequest{
			Attempt:     attemptNo,
			Status:      status,
			AsynqTaskID: asynqID,
			Error:       errMsg,
			WorkerName:  w.workerName,
			StartedAt:   &start,
			FinishedAt:  &finished,
			DurationMs:  &dur,
		})

		return err
	})
}

// Enqueue：通过控制面入队任务
func (w *Worker) Enqueue(queueName string, task *Task) {
	if task == nil {
		log.Printf("enqueue ignored: nil task")
		return
	}
	if task.TaskID == "" {
		task.TaskID = task.ID
	}

	// 通过控制面入队（保证落库并能在前端展示）
	if w.baseURL != "" {
		delaySeconds := int32(0)
		if w.defaultDelay > 0 {
			delaySeconds = int32(w.defaultDelay.Seconds())
		}
		_, err := NewClient(w.baseURL).EnqueueTask(context.Background(), EnqueueTaskRequest{
			WorkerName:   w.workerName,
			Queue:        queueName,
			TaskID:       task.TaskID,
			Payload:      task.Payload,
			DelaySeconds: int(delaySeconds),
		})
		if err != nil {
			log.Printf("enqueue via control-plane failed: %v", err)
		}
		return
	}

	// fallback：无 baseURL 时直连 Redis（不建议）
	fullQueueName := w.queueName(queueName)
	b, _ := json.Marshal(task)
	opts := []asynq.Option{
		asynq.Queue(fullQueueName),
		asynq.MaxRetry(w.defaultRetryCount),
		asynq.Timeout(w.defaultTimeout),
	}
	if w.defaultDelay > 0 {
		opts = append(opts, asynq.ProcessIn(w.defaultDelay))
	}
	_, err := w.client.Enqueue(asynq.NewTask(fullQueueName, b), opts...)
	if err != nil {
		log.Printf("enqueue failed: %v", err)
	}
}

func (w *Worker) Run() error {
	if w.server == nil || w.client == nil {
		return errors.New("worker not initialized")
	}

	// 注册 worker 到控制面
	if w.autoRegister && !w.registeredOnce {
		config := WorkerConfig{
			WorkerName:        w.workerName,
			BaseURL:           w.baseURL,
			RedisAddr:         w.redisURI,
			Concurrency:       w.concurrency,
			Queues:            w.queues,
			DefaultRetryCount: w.defaultRetryCount,
			DefaultTimeout:    int(w.defaultTimeout.Seconds()),
			DefaultDelay:      int(w.defaultDelay.Seconds()),
		}
		_ = w.registrar.RegisterWorker(context.Background(), config, w.overwriteReg)
		w.registeredOnce = true
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		w.server.Stop()
		w.client.Close()
	}()

	log.Printf("asynqhub-worker start: workerName=%s redis=%s concurrency=%d baseURL=%s", w.workerName, w.redisURI, w.concurrency, w.baseURL)
	return w.server.Run(w.mux)
}

// DefaultWorkerName 默认 worker 名（用于日志与上报）。
func DefaultWorkerName() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
