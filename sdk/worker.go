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

// 默认优先级权重
const (
	PriorityCritical = "critical"
	PriorityDefault  = "default"
	PriorityLow      = "low"
)

// DefaultPriorities 默认优先级权重配置
var DefaultPriorities = map[string]int{
	PriorityCritical: 50,
	PriorityDefault:  30,
	PriorityLow:      10,
}

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

// QueueGroupConfig 队列组配置（用户传入）
type QueueGroupConfig struct {
	Name        string         `json:"name"`        // 队列组名称（例如：web_crawl）
	Concurrency int            `json:"concurrency"` // 并发数
	Priorities  map[string]int `json:"priorities"`  // 优先级权重，可选，默认为 critical=50, default=30, low=10
}

// QueueGroup 队列组（内部使用）
type QueueGroup struct {
	Name        string              // 队列组名称
	Concurrency int                 // 并发数
	Priorities  map[string]int      // 优先级权重
	Server      *asynq.Server       // 独立的 Server 实例
	Mux         *asynq.ServeMux     // 独立的 ServeMux
	handlers    map[string]struct{} // 已注册的处理器
}

// Worker：队列组架构
//
// 新架构特点：
// - 一个 Worker 管理多个队列组（QueueGroup）
// - 每个队列组对应一个独立的 asynq.Server 实例
// - 每个队列组默认包含 3 个优先级队列：critical、default、low
// - 队列命名：workerName:queueGroupName:priority
type Worker struct {
	workerName string

	baseURL  string
	redisURI string
	redisOpt asynq.RedisClientOpt

	queueGroups map[string]*QueueGroup // 队列组映射

	defaultRetryCount int
	defaultTimeout    time.Duration
	defaultDelay      time.Duration

	autoRegister bool
	overwriteReg bool

	client *asynq.Client

	reporter  Reporter
	registrar Registrar

	registeredOnce bool
}

type Option func(*Worker)

func WithBaseURL(baseURL string) Option { return func(w *Worker) { w.baseURL = baseURL } }
func WithRedisAddr(redis string) Option {
	return func(w *Worker) {
		w.redisURI = redis
	}
}
func WithDefaultRetryCount(n int) Option        { return func(w *Worker) { w.defaultRetryCount = n } }
func WithDefaultTimeout(d time.Duration) Option { return func(w *Worker) { w.defaultTimeout = d } }
func WithDefaultDelay(d time.Duration) Option   { return func(w *Worker) { w.defaultDelay = d } }
func WithoutAutoRegister() Option               { return func(w *Worker) { w.autoRegister = false } }
func WithRegisterOverwrite() Option             { return func(w *Worker) { w.overwriteReg = true } }

// WithQueueGroups 配置队列组
func WithQueueGroups(groups []QueueGroupConfig) Option {
	return func(w *Worker) {
		for _, g := range groups {
			qg := &QueueGroup{
				Name:        g.Name,
				Concurrency: g.Concurrency,
				Priorities:  g.Priorities,
				handlers:    make(map[string]struct{}),
			}
			// 设置默认优先级权重
			if len(qg.Priorities) == 0 {
				qg.Priorities = make(map[string]int)
				for k, v := range DefaultPriorities {
					qg.Priorities[k] = v
				}
			}
			// 设置默认并发数
			if qg.Concurrency <= 0 {
				qg.Concurrency = 10
			}
			w.queueGroups[g.Name] = qg
		}
	}
}

// New：创建 Worker
// 第一个参数 workerName（用于日志与上报）
func New(workerName string, opts ...Option) (*Worker, error) {
	w := &Worker{
		workerName:        workerName,
		redisURI:          os.Getenv("REDIS_ADDR"),
		queueGroups:       make(map[string]*QueueGroup),
		defaultRetryCount: 3,
		defaultTimeout:    30 * time.Second,
		defaultDelay:      0,
		autoRegister:      true,
		overwriteReg:      false,
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

	if w.workerName == "" {
		w.workerName = DefaultWorkerName()
	}

	// 如果没有配置队列组，创建一个默认的队列组
	if len(w.queueGroups) == 0 {
		w.queueGroups["default"] = &QueueGroup{
			Name:        "default",
			Concurrency: 10,
			Priorities:  DefaultPriorities,
			handlers:    make(map[string]struct{}),
		}
	}

	// 为每个队列组创建独立的 Server 实例
	for name, qg := range w.queueGroups {
		// 构建队列配置（包含 3 个优先级）
		queues := make(map[string]int)
		for priority, weight := range qg.Priorities {
			fullQueueName := w.fullQueueName(name, priority)
			queues[fullQueueName] = weight
		}

		// 创建独立的 Server 和 Mux
		qg.Server = asynq.NewServer(w.redisOpt, asynq.Config{
			Concurrency: qg.Concurrency,
			Queues:      queues,
		})
		qg.Mux = asynq.NewServeMux()
	}

	w.reporter = Reporter{
		ControlPlaneURL: w.baseURL,
		WorkerName:      w.workerName,
	}
	w.registrar = Registrar{
		ControlPlaneURL: w.baseURL,
	}

	w.client = asynq.NewClient(w.redisOpt)

	return w, nil
}

// fullQueueName 生成完整队列名：workerName:queueGroupName:priority
func (w *Worker) fullQueueName(queueGroup, priority string) string {
	return fmt.Sprintf("%s:%s:%s", w.workerName, queueGroup, priority)
}

// parseQueueName 解析队列名，返回队列组名和优先级
func (w *Worker) parseQueueName(queueGroup string) (string, string) {
	// 如果包含优先级后缀，解析出来
	parts := strings.Split(queueGroup, ":")
	if len(parts) >= 2 {
		// 检查最后一个部分是否是优先级
		lastPart := parts[len(parts)-1]
		if lastPart == PriorityCritical || lastPart == PriorityDefault || lastPart == PriorityLow {
			return strings.Join(parts[:len(parts)-1], ":"), lastPart
		}
	}
	return queueGroup, PriorityDefault
}

// HandleFunc 为队列组注册处理器（处理所有优先级）
// SDK 会在外层自动上报 attempt/状态。
func (w *Worker) HandleFunc(queueGroup string, fn func(ctx context.Context, t *asynq.Task) error) {
	qg, ok := w.queueGroups[queueGroup]
	if !ok {
		log.Printf("警告: 队列组 %s 不存在，忽略注册", queueGroup)
		return
	}

	// 为所有优先级注册相同的处理器
	for priority := range qg.Priorities {
		w.registerHandler(qg, queueGroup, priority, fn)
	}
}

// HandleFuncWithPriority 为队列组的特定优先级注册处理器
func (w *Worker) HandleFuncWithPriority(queueGroup, priority string, fn func(ctx context.Context, t *asynq.Task) error) {
	qg, ok := w.queueGroups[queueGroup]
	if !ok {
		log.Printf("警告: 队列组 %s 不存在，忽略注册", queueGroup)
		return
	}

	if _, ok := qg.Priorities[priority]; !ok {
		log.Printf("警告: 优先级 %s 不存在于队列组 %s 中，忽略注册", priority, queueGroup)
		return
	}

	w.registerHandler(qg, queueGroup, priority, fn)
}

// registerHandler 内部方法：注册处理器
func (w *Worker) registerHandler(qg *QueueGroup, queueGroup, priority string, fn func(ctx context.Context, t *asynq.Task) error) {
	fullQueueName := w.fullQueueName(queueGroup, priority)

	// 检查是否已注册
	if _, exists := qg.handlers[fullQueueName]; exists {
		return
	}
	qg.handlers[fullQueueName] = struct{}{}

	qg.Mux.HandleFunc(fullQueueName, func(ctx context.Context, t *asynq.Task) error {
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

// Enqueue 入队到默认优先级（default）
func (w *Worker) Enqueue(queueGroup string, task *Task) {
	w.EnqueueWithPriority(queueGroup, PriorityDefault, task)
}

// EnqueueWithPriority 入队到指定优先级
func (w *Worker) EnqueueWithPriority(queueGroup, priority string, task *Task) {
	if task == nil {
		log.Printf("enqueue ignored: nil task")
		return
	}
	if task.TaskID == "" {
		task.TaskID = task.ID
	}

	// 验证队列组和优先级
	qg, ok := w.queueGroups[queueGroup]
	if !ok {
		log.Printf("enqueue ignored: queue group %s not found", queueGroup)
		return
	}
	if _, ok := qg.Priorities[priority]; !ok {
		log.Printf("enqueue ignored: priority %s not found in queue group %s", priority, queueGroup)
		return
	}

	// 通过控制面入队（保证落库并能在前端展示）
	if w.baseURL != "" {
		delaySeconds := int32(0)
		if w.defaultDelay > 0 {
			delaySeconds = int32(w.defaultDelay.Seconds())
		}
		_, err := NewClient(w.baseURL).EnqueueTask(context.Background(), EnqueueTaskRequest{
			WorkerName:   w.workerName,
			Queue:        queueGroup,
			Priority:     priority,
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
	fullQueueName := w.fullQueueName(queueGroup, priority)
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

// Run 启动 Worker
func (w *Worker) Run() error {
	if w.client == nil {
		return errors.New("worker not initialized")
	}

	if len(w.queueGroups) == 0 {
		return errors.New("no queue groups configured")
	}

	// 注册 worker 到控制面
	if w.autoRegister && !w.registeredOnce {
		config := w.buildWorkerConfig()
		_ = w.registrar.RegisterWorker(context.Background(), config, w.overwriteReg)
		w.registeredOnce = true
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, len(w.queueGroups))
	var wg sync.WaitGroup

	// 启动所有队列组的 Server
	for name, qg := range w.queueGroups {
		wg.Add(1)
		go func(name string, qg *QueueGroup) {
			defer wg.Done()
			log.Printf("启动队列组: %s (并发数: %d, 优先级: %v)", name, qg.Concurrency, qg.Priorities)
			if err := qg.Server.Run(qg.Mux); err != nil {
				errChan <- fmt.Errorf("queue group %s: %w", name, err)
			}
		}(name, qg)
	}

	log.Printf("asynqhub-worker 启动: workerName=%s redis=%s queueGroups=%d baseURL=%s",
		w.workerName, w.redisURI, len(w.queueGroups), w.baseURL)

	// 等待信号或错误
	select {
	case <-ctx.Done():
		log.Printf("收到终止信号，正在关闭...")
		w.shutdown()
		wg.Wait()
		return nil
	case err := <-errChan:
		w.shutdown()
		wg.Wait()
		return err
	}
}

// shutdown 关闭所有资源
func (w *Worker) shutdown() {
	for name, qg := range w.queueGroups {
		log.Printf("关闭队列组: %s", name)
		qg.Server.Shutdown()
	}
	if w.client != nil {
		w.client.Close()
	}
}

// buildWorkerConfig 构建 Worker 配置（用于注册到控制面）
func (w *Worker) buildWorkerConfig() WorkerConfig {
	queueGroups := make([]QueueGroupConfig, 0, len(w.queueGroups))
	for _, qg := range w.queueGroups {
		queueGroups = append(queueGroups, QueueGroupConfig{
			Name:        qg.Name,
			Concurrency: qg.Concurrency,
			Priorities:  qg.Priorities,
		})
	}

	return WorkerConfig{
		WorkerName:        w.workerName,
		BaseURL:           w.baseURL,
		RedisAddr:         w.redisURI,
		QueueGroups:       queueGroups,
		DefaultRetryCount: w.defaultRetryCount,
		DefaultTimeout:    int(w.defaultTimeout.Seconds()),
		DefaultDelay:      int(w.defaultDelay.Seconds()),
	}
}

// GetQueueGroups 获取队列组配置（用于调试）
func (w *Worker) GetQueueGroups() map[string]*QueueGroup {
	return w.queueGroups
}

// DefaultWorkerName 默认 worker 名（用于日志与上报）。
func DefaultWorkerName() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
