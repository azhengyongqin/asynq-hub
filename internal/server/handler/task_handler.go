package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/azhengyongqin/asynq-hub/internal/middleware"
	"github.com/azhengyongqin/asynq-hub/internal/model"
	asynqx "github.com/azhengyongqin/asynq-hub/internal/queue"
	"github.com/azhengyongqin/asynq-hub/internal/repository"
	"github.com/azhengyongqin/asynq-hub/internal/server/dto"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// TaskHandler Task 相关 API Handler
type TaskHandler struct {
	asynqClient *asynq.Client
	taskRepo    repository.TaskRepository
	workerStore *workers.Store
}

// NewTaskHandler 创建 TaskHandler
func NewTaskHandler(asynqClient *asynq.Client, taskRepo repository.TaskRepository, workerStore *workers.Store) *TaskHandler {
	return &TaskHandler{
		asynqClient: asynqClient,
		taskRepo:    taskRepo,
		workerStore: workerStore,
	}
}

// CreateTask godoc
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
func (h *TaskHandler) CreateTask(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 验证 worker_name 格式
	if !middleware.ValidateWorkerName(req.WorkerName) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "worker_name 格式无效"})
		return
	}

	// 验证 queue 格式
	if !middleware.ValidateQueueName(req.Queue) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "queue 格式无效"})
		return
	}

	// 验证 task_id 格式（如果提供）
	if req.TaskID != "" && !middleware.ValidateTaskID(req.TaskID) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "task_id 格式无效"})
		return
	}

	// 验证 payload 大小
	if len(req.Payload) > middleware.MaxPayloadSize {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "payload 过大，最大 2MB"})
		return
	}

	// 验证 worker 是否存在
	workerCfg, ok := h.workerStore.Get(req.WorkerName)
	if !ok {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}
	if !workerCfg.IsEnabled {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "worker 未启用"})
		return
	}

	// 验证 queue 是否在 worker 的配置中
	if !h.workerStore.HasQueue(req.WorkerName, req.Queue) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "队列不存在于 worker 配置中"})
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

	info, err := h.asynqClient.Enqueue(t, asynqx.EnqueueOptions(p)...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 记录到数据库
	if h.taskRepo != nil {
		_ = h.taskRepo.UpsertTask(c.Request.Context(), repository.Task{
			TaskID:      taskID,
			WorkerName:  req.WorkerName,
			Queue:       fullQueue,
			Priority:    0,
			Payload:     req.Payload,
			Status:      string(model.TaskStatusPending),
			LastAttempt: 0,
		})
	}

	c.JSON(http.StatusOK, dto.CreateTaskResponse{
		TaskID:      taskID,
		WorkerName:  req.WorkerName,
		Queue:       req.Queue,
		AsynqTaskID: info.ID,
		Status:      "enqueued",
	})
}

// ListTasks godoc
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
func (h *TaskHandler) ListTasks(c *gin.Context) {
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.taskRepo.ListTasks(c.Request.Context(), repository.ListTasksFilter{
		WorkerName: c.DefaultQuery("worker_name", ""),
		Status:     c.DefaultQuery("status", ""),
		Queue:      c.DefaultQuery("queue", ""),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}
	total, err := h.taskRepo.CountTasks(c.Request.Context(), repository.ListTasksFilter{
		WorkerName: c.DefaultQuery("worker_name", ""),
		Status:     c.DefaultQuery("status", ""),
		Queue:      c.DefaultQuery("queue", ""),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.TaskListResponse{Items: items, Total: total})
}

// GetTask godoc
// @Summary 获取任务详情
// @Description 根据 task_id 获取任务详细信息及执行历史
// @Tags Tasks
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} dto.TaskResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 501 {object} dto.ErrorResponse
// @Router /tasks/{task_id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	taskID := c.Param("task_id")
	t, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "task 不存在"})
		return
	}
	attempts, _ := h.taskRepo.ListAttempts(c.Request.Context(), taskID, 50)
	c.JSON(http.StatusOK, gin.H{"item": t, "attempts": attempts})
}

// ReplayTask godoc
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
func (h *TaskHandler) ReplayTask(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
		return
	}
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	taskID := c.Param("task_id")

	var req dto.ReplayTaskRequest
	_ = c.ShouldBindJSON(&req)

	t, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "task 不存在"})
		return
	}

	if t.Status != string(model.TaskStatusSuccess) && t.Status != string(model.TaskStatusFail) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "只能重放已结束的任务（success/fail）"})
		return
	}

	workerCfg, ok := h.workerStore.Get(t.WorkerName)
	if !ok {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}

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

	task := asynq.NewTask(t.Queue, b)
	p := asynqx.EnqueueParams{
		TaskType:       t.Queue,
		TaskKey:        newTaskID,
		Queue:          t.Queue,
		MaxRetry:       workerCfg.DefaultRetryCount,
		TimeoutSeconds: workerCfg.DefaultTimeout,
		DelaySeconds:   int32(req.Delay),
		Payload:        t.Payload,
	}

	_, err = h.asynqClient.Enqueue(task, asynqx.EnqueueOptions(p)...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	_ = h.taskRepo.UpsertTask(c.Request.Context(), repository.Task{
		TaskID:      newTaskID,
		WorkerName:  t.WorkerName,
		Queue:       t.Queue,
		Priority:    t.Priority,
		Payload:     t.Payload,
		Status:      string(model.TaskStatusPending),
		LastAttempt: 0,
	})

	c.JSON(http.StatusOK, dto.ReplayTaskResponse{
		Status:    "replayed",
		NewTaskID: newTaskID,
	})
}

// ReportAttempt godoc
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
func (h *TaskHandler) ReportAttempt(c *gin.Context) {
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	taskID := c.Param("task_id")
	var req dto.ReportAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "task 不存在"})
		return
	}

	attemptStatus := model.TaskStatusRunning
	switch req.Status {
	case "running":
		attemptStatus = model.TaskStatusRunning
	case "success":
		attemptStatus = model.TaskStatusSuccess
	case "fail":
		attemptStatus = model.TaskStatusFail
	default:
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "status 必须是 running/success/fail"})
		return
	}

	attempt := repository.Attempt{
		TaskID:     taskID,
		Attempt:    req.Attempt,
		Status:     string(attemptStatus),
		StartedAt:  req.StartedAt,
		FinishedAt: req.FinishedAt,
		Error:      req.Error,
		TraceID:    req.TraceID,
		SpanID:     req.SpanID,
	}

	if err := h.taskRepo.InsertAttempt(c.Request.Context(), attempt); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if req.Status == "success" || req.Status == "fail" {
		task.Status = string(attemptStatus)
		task.LastAttempt = req.Attempt
		if err := h.taskRepo.UpsertTask(c.Request.Context(), *task); err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{Status: "ok", Message: "上报成功"})
}

// BatchRetry godoc
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
func (h *TaskHandler) BatchRetry(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
		return
	}
	if h.taskRepo == nil {
		c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Error: "Postgres 未配置"})
		return
	}

	var req dto.BatchRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	var taskIDs []string
	if len(req.TaskIDs) > 0 {
		taskIDs = req.TaskIDs
	} else {
		filter := repository.ListTasksFilter{
			WorkerName: req.WorkerName,
			Status:     req.Status,
			Limit:      limit,
		}
		tasks, err := h.taskRepo.ListTasks(c.Request.Context(), filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
		for _, t := range tasks {
			taskIDs = append(taskIDs, t.TaskID)
		}
	}

	var newTaskIDs []string
	for _, oldTaskID := range taskIDs {
		t, err := h.taskRepo.GetTask(c.Request.Context(), oldTaskID)
		if err != nil {
			continue
		}

		workerCfg, ok := h.workerStore.Get(t.WorkerName)
		if !ok {
			continue
		}

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

		task := asynq.NewTask(t.Queue, b)
		p := asynqx.EnqueueParams{
			TaskType:       t.Queue,
			TaskKey:        newTaskID,
			Queue:          t.Queue,
			MaxRetry:       workerCfg.DefaultRetryCount,
			TimeoutSeconds: workerCfg.DefaultTimeout,
			Payload:        t.Payload,
		}

		if _, err := h.asynqClient.Enqueue(task, asynqx.EnqueueOptions(p)...); err != nil {
			continue
		}

		_ = h.taskRepo.UpsertTask(c.Request.Context(), repository.Task{
			TaskID:      newTaskID,
			WorkerName:  t.WorkerName,
			Queue:       t.Queue,
			Priority:    t.Priority,
			Payload:     t.Payload,
			Status:      string(model.TaskStatusPending),
			LastAttempt: 0,
		})

		newTaskIDs = append(newTaskIDs, newTaskID)
	}

	c.JSON(http.StatusOK, dto.BatchRetryResponse{
		Status:       "ok",
		TotalRetried: len(newTaskIDs),
		NewTaskIDs:   newTaskIDs,
	})
}
