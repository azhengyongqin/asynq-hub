package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azhengyongqin/asynq-hub/sdk"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

// TestDataGenerator 测试数据生成器
// 用于生成大量测试任务，测试 Workers 面板的监控和统计能力
func main() {
	if err := loadEnvFile(); err != nil {
		log.Printf("警告: 无法加载 .env 文件: %v（将使用环境变量或默认值）", err)
	}

	// 检查是否是测试数据生成模式
	if len(os.Args) > 1 && os.Args[1] == "generate-test-data" {
		generateTestData()
		return
	}

	// 原始 worker 逻辑
	runWorker()
}

// generateTestData 生成测试数据
func generateTestData() {
	log.Println("=== 开始生成测试数据 ===")

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:28080"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis://localhost:16379/0"
	}

	// 创建 3 个不同的 worker 配置（使用新的队列组结构）
	workers := []struct {
		name        string
		queueGroups []sdk.QueueGroupConfig
	}{
		{
			name: "crawler-worker-1",
			queueGroups: []sdk.QueueGroupConfig{
				{Name: "web_crawl", Concurrency: 10},
				{Name: "api_crawl", Concurrency: 8},
				{Name: "image_crawl", Concurrency: 5},
			},
		},
		{
			name: "ai-worker-1",
			queueGroups: []sdk.QueueGroupConfig{
				{Name: "prompt_crawl", Concurrency: 10},
				{Name: "text_analyze", Concurrency: 8},
				{Name: "image_process", Concurrency: 6},
			},
		},
		{
			name: "data-worker-1",
			queueGroups: []sdk.QueueGroupConfig{
				{Name: "data_process", Concurrency: 10},
				{Name: "data_export", Concurrency: 7},
				{Name: "data_import", Concurrency: 5},
			},
		},
	}

	// 注册所有 workers
	for _, w := range workers {
		worker, err := sdk.New(
			w.name,
			sdk.WithBaseURL(baseURL),
			sdk.WithRedisAddr(redisAddr),
			sdk.WithDefaultRetryCount(3),
			sdk.WithDefaultTimeout(30*time.Second),
			sdk.WithQueueGroups(w.queueGroups),
			sdk.WithRegisterOverwrite(), // 覆盖已有配置以确保队列配置正确
		)
		if err != nil {
			log.Fatalf("创建 worker %s 失败: %v", w.name, err)
		}

		// 等待确保注册完成
		time.Sleep(500 * time.Millisecond)

		log.Printf("✓ 注册 worker: %s (队列组: %v)", w.name, getQueueGroupNames(w.queueGroups))

		// 为每个 worker 生成测试任务
		generateTasksForWorker(worker, w.name, w.queueGroups)

		log.Println()
	}

	log.Println("=== 测试数据生成完成 ===")
	log.Println("提示: 启动 worker 来处理这些任务:")
	log.Println("  go run ./cmd/worker")
}

// generateTasksForWorker 为指定 worker 生成测试任务
func generateTasksForWorker(worker *sdk.Worker, workerName string, queueGroups []sdk.QueueGroupConfig) {
	rand.Seed(time.Now().UnixNano())

	// 每个队列组生成不同数量的任务
	taskCounts := map[string]int{
		"web_crawl":     50,
		"api_crawl":     30,
		"image_crawl":   20,
		"prompt_crawl":  40,
		"text_analyze":  35,
		"image_process": 25,
		"data_process":  45,
		"data_export":   15,
		"data_import":   20,
	}

	priorities := []string{sdk.PriorityCritical, sdk.PriorityDefault, sdk.PriorityLow}

	for _, qg := range queueGroups {
		count := taskCounts[qg.Name]
		if count == 0 {
			count = 20 // 默认生成 20 个任务
		}

		for i := 0; i < count; i++ {
			taskID := fmt.Sprintf("%s-%s-%d-%d", workerName, qg.Name, time.Now().Unix(), i)
			payload := generatePayloadForQueue(qg.Name, i)

			// 随机选择优先级
			priority := priorities[rand.Intn(len(priorities))]

			worker.EnqueueWithPriority(qg.Name, priority, &sdk.Task{
				TaskID:  taskID,
				Payload: payload,
			})

			// 避免过快入队
			time.Sleep(5 * time.Millisecond)
		}
		log.Printf("  → 队列组 %s: 已生成 %d 个任务（分布在 critical/default/low 三个优先级）", qg.Name, count)
	}
}

// generatePayloadForQueue 根据队列类型生成合适的 payload
func generatePayloadForQueue(queueName string, index int) []byte {
	rand.Seed(time.Now().UnixNano() + int64(index))

	// 20% 的任务故意设置为会失败的 payload
	willFail := rand.Float32() < 0.2

	switch queueName {
	case "web_crawl":
		url := fmt.Sprintf("https://example.com/page-%d", index)
		if willFail {
			url = fmt.Sprintf("fail-https://example.com/page-%d", index)
		}
		return []byte(fmt.Sprintf(`{"url":"%s"}`, url))

	case "api_crawl":
		endpoint := fmt.Sprintf("https://api.example.com/v1/data/%d", index)
		if willFail {
			endpoint = "" // 空 endpoint 会失败
		}
		return []byte(fmt.Sprintf(`{"endpoint":"%s","method":"GET"}`, endpoint))

	case "image_crawl":
		imageURL := fmt.Sprintf("https://cdn.example.com/images/img-%d.jpg", index)
		if willFail {
			imageURL = fmt.Sprintf("fail-%s", imageURL)
		}
		return []byte(fmt.Sprintf(`{"image_url":"%s","size":"1920x1080"}`, imageURL))

	case "prompt_crawl":
		url := fmt.Sprintf("https://example.com/article-%d", index)
		prompt := "请抽取标题、摘要、作者、发布时间"
		if willFail {
			prompt = "" // 空 prompt 会失败
		}
		return []byte(fmt.Sprintf(`{"url":"%s","prompt":"%s"}`, url, prompt))

	case "text_analyze":
		text := fmt.Sprintf("这是第 %d 条待分析的文本内容...", index)
		if willFail {
			text = "fail-" + text
		}
		return []byte(fmt.Sprintf(`{"text":"%s","analysis_type":"sentiment"}`, text))

	case "image_process":
		imageID := fmt.Sprintf("img-%d", index)
		operation := "resize"
		if willFail {
			operation = "fail-resize"
		}
		return []byte(fmt.Sprintf(`{"image_id":"%s","operation":"%s","params":{"width":800,"height":600}}`, imageID, operation))

	case "data_process":
		dataID := fmt.Sprintf("data-%d", index)
		processType := "transform"
		if willFail {
			processType = ""
		}
		return []byte(fmt.Sprintf(`{"data_id":"%s","process_type":"%s"}`, dataID, processType))

	case "data_export":
		exportID := fmt.Sprintf("export-%d", index)
		format := "csv"
		if willFail {
			format = "fail-csv"
		}
		return []byte(fmt.Sprintf(`{"export_id":"%s","format":"%s"}`, exportID, format))

	case "data_import":
		importID := fmt.Sprintf("import-%d", index)
		source := "s3://bucket/data.json"
		if willFail {
			source = ""
		}
		return []byte(fmt.Sprintf(`{"import_id":"%s","source":"%s"}`, importID, source))

	default:
		return []byte(fmt.Sprintf(`{"task_index":%d}`, index))
	}
}

// runWorker 运行 worker（原始逻辑）
func runWorker() {
	workerName := os.Getenv("WORKER_NAME")
	if workerName == "" {
		workerName = "worker-1"
	}

	work, err := sdk.New(
		workerName,
		sdk.WithBaseURL(os.Getenv("BASE_URL")),
		sdk.WithRedisAddr(os.Getenv("REDIS_ADDR")),
		sdk.WithDefaultRetryCount(3),
		sdk.WithDefaultTimeout(30*time.Second),
		sdk.WithDefaultDelay(10*time.Second),
		sdk.WithQueueGroups([]sdk.QueueGroupConfig{
			{Name: "web_crawl", Concurrency: 10},
			{Name: "prompt_crawl", Concurrency: 10},
			{Name: "api_crawl", Concurrency: 8},
			{Name: "image_crawl", Concurrency: 7},
			{Name: "text_analyze", Concurrency: 9},
			{Name: "image_process", Concurrency: 8},
			{Name: "data_process", Concurrency: 10},
			{Name: "data_export", Concurrency: 6},
			{Name: "data_import", Concurrency: 7},
		}),
	)
	if err != nil {
		log.Fatalf("new worker: %v", err)
	}

	// 注册所有任务处理器
	registerHandlers(work)

	// 启动时创建几个测试任务
	createInitialTasks(work)

	if err := work.Run(); err != nil {
		log.Fatalf("worker run: %v", err)
	}
}

// registerHandlers 注册所有任务处理器
func registerHandlers(work *sdk.Worker) {
	// web_crawl: 网页抓取（为整个队列组注册处理器）
	work.HandleFunc("web_crawl", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "web_crawl", 200, 400)
	})

	// api_crawl: API 抓取
	work.HandleFunc("api_crawl", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "api_crawl", 100, 300)
	})

	// image_crawl: 图片抓取
	work.HandleFunc("image_crawl", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "image_crawl", 300, 600)
	})

	// prompt_crawl: Prompt 处理
	work.HandleFunc("prompt_crawl", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "prompt_crawl", 400, 800)
	})

	// text_analyze: 文本分析
	work.HandleFunc("text_analyze", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "text_analyze", 250, 500)
	})

	// image_process: 图片处理
	work.HandleFunc("image_process", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "image_process", 350, 700)
	})

	// data_process: 数据处理
	work.HandleFunc("data_process", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "data_process", 150, 400)
	})

	// data_export: 数据导出
	work.HandleFunc("data_export", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "data_export", 500, 1000)
	})

	// data_import: 数据导入
	work.HandleFunc("data_import", func(ctx context.Context, t *asynq.Task) error {
		return handleTask(ctx, t, "data_import", 600, 1200)
	})
}

// handleTask 通用任务处理函数
func handleTask(ctx context.Context, t *asynq.Task, taskType string, minMs, maxMs int) error {
	task := sdk.Task{}
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return err
	}

	var payload map[string]interface{}
	if err := task.UnmarshalPayload(&payload); err != nil {
		return err
	}

	log.Printf("[%s] 开始: task_id=%s", taskType, task.TaskID)

	// 模拟处理时间（随机）
	duration := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	time.Sleep(duration)

	// 检查是否有 "always_fail" 标记（用于模拟超过最大重试次数）
	if alwaysFail, ok := payload["always_fail"].(bool); ok && alwaysFail {
		log.Printf("[%s] 持续失败: task_id=%s (always_fail=true, 将超过最大重试次数)", taskType, task.TaskID)
		return fmt.Errorf("任务配置为持续失败: always_fail=true")
	}

	// 检查是否应该失败
	for _, v := range payload {
		if str, ok := v.(string); ok {
			if strings.HasPrefix(str, "fail") || str == "" {
				log.Printf("[%s] 失败: task_id=%s (故意失败)", taskType, task.TaskID)
				return fmt.Errorf("任务故意失败: %v", payload)
			}
		}
	}

	log.Printf("[%s] 完成: task_id=%s (耗时: %v)", taskType, task.TaskID, duration)
	return nil
}

// createInitialTasks 创建初始测试任务
func createInitialTasks(work *sdk.Worker) {
	suffix := fmt.Sprintf("%d", time.Now().Unix())

	tasks := []struct {
		queueGroup string
		priority   string
		payload    string
	}{
		{"web_crawl", sdk.PriorityCritical, `{"url":"https://example.com"}`},
		{"prompt_crawl", sdk.PriorityDefault, `{"url":"https://example.com","prompt":"请抽取标题、摘要、发布时间"}`},
		{"api_crawl", sdk.PriorityLow, `{"endpoint":"https://api.example.com/v1/users","method":"GET"}`},
		{"image_crawl", sdk.PriorityDefault, `{"image_url":"https://cdn.example.com/test.jpg","size":"1920x1080"}`},
	}

	for i, task := range tasks {
		taskID := fmt.Sprintf("%s-%s-%d", task.queueGroup, suffix, i)
		work.EnqueueWithPriority(task.queueGroup, task.priority, &sdk.Task{
			TaskID:  taskID,
			Payload: []byte(task.payload),
		})
	}
}

// loadEnvFile 尝试从项目根目录加载 .env 文件
func loadEnvFile() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	possiblePaths := []string{
		filepath.Join(wd, ".env"),
		filepath.Join(wd, "..", "..", ".env"),
		filepath.Join(wd, "..", ".env"),
		filepath.Join(wd, "..", "..", "..", ".env"),
	}

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

	if envPath == "" {
		return nil
	}

	if err := godotenv.Load(envPath); err != nil {
		return err
	}

	log.Printf("已加载环境变量文件: %s", envPath)
	return nil
}

// getQueueGroupNames 获取队列组名列表
func getQueueGroupNames(queueGroups []sdk.QueueGroupConfig) []string {
	names := make([]string, len(queueGroups))
	for i, qg := range queueGroups {
		names[i] = qg.Name
	}
	return names
}
