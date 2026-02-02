package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/azhengyongqin/asynq-hub/internal/server/dto"
	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// QueueHandler Queue 相关 API Handler
type QueueHandler struct {
	asynqClient *asynq.Client
	workerStore *workers.Store
}

// NewQueueHandler 创建 QueueHandler
func NewQueueHandler(asynqClient *asynq.Client, workerStore *workers.Store) *QueueHandler {
	return &QueueHandler{
		asynqClient: asynqClient,
		workerStore: workerStore,
	}
}

// GetQueueStats godoc
// @Summary 查询队列状态
// @Description 获取指定 Worker 的所有队列状态统计
// @Tags Queues
// @Produce json
// @Param worker_name query string true "Worker 名称"
// @Success 200 {object} dto.QueueStatsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /queues/stats [get]
func (h *QueueHandler) GetQueueStats(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
		return
	}

	workerName := c.Query("worker_name")
	if workerName == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "worker_name 参数必填"})
		return
	}

	// 获取 worker 配置
	workerCfg, ok := h.workerStore.Get(workerName)
	if !ok {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})

	var queues []interface{}
	// 遍历所有队列组和优先级
	for _, qg := range workerCfg.QueueGroups {
		for priority := range qg.Priorities {
			fullQueue := workerCfg.FullQueueName(qg.Name, priority)
			info, err := inspector.GetQueueInfo(fullQueue)
			if err != nil {
				continue
			}

			queues = append(queues, gin.H{
				"queue_group": qg.Name,
				"priority":    priority,
				"full_name":   fullQueue,
				"pending":     info.Pending,
				"active":      info.Active,
				"scheduled":   info.Scheduled,
				"retry":       info.Retry,
				"archived":    info.Archived,
				"completed":   info.Completed,
				"size":        info.Size,
			})
		}
	}

	c.JSON(http.StatusOK, dto.QueueStatsResponse{
		WorkerName: workerName,
		Queues:     queues,
	})
}

// ClearQueue godoc
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
func (h *QueueHandler) ClearQueue(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
		return
	}

	var req dto.ClearQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	workerCfg, ok := h.workerStore.Get(req.WorkerName)
	if !ok {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})

	var clearedQueues []string
	totalDeleted := 0

	if req.QueueName != "" {
		// 清空指定队列组的所有优先级
		qg := workerCfg.GetQueueGroup(req.QueueName)
		if qg != nil {
			for priority := range qg.Priorities {
				fullQueue := workerCfg.FullQueueName(req.QueueName, priority)
				deleted, err := inspector.DeleteAllPendingTasks(fullQueue)
				if err == nil {
					clearedQueues = append(clearedQueues, fullQueue)
					totalDeleted += deleted
				}
			}
		}
	} else {
		// 清空所有队列组的所有优先级
		for _, qg := range workerCfg.QueueGroups {
			for priority := range qg.Priorities {
				fullQueue := workerCfg.FullQueueName(qg.Name, priority)

				// 删除 pending 任务
				deleted, err := inspector.DeleteAllPendingTasks(fullQueue)
				if err != nil {
					continue
				}

				clearedQueues = append(clearedQueues, fullQueue)
				totalDeleted += deleted
			}
		}
	}

	c.JSON(http.StatusOK, dto.ClearQueueResponse{
		Status:        "ok",
		WorkerName:    req.WorkerName,
		ClearedQueues: clearedQueues,
		TotalDeleted:  totalDeleted,
	})
}

// ClearDeadQueue godoc
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
func (h *QueueHandler) ClearDeadQueue(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "asynq client 未配置"})
		return
	}

	var req dto.ClearDeadQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	workerCfg, ok := h.workerStore.Get(req.WorkerName)
	if !ok {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "worker 不存在"})
		return
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: workerCfg.RedisAddr})

	var clearedQueues []string
	totalDeleted := 0

	if req.QueueName != "" {
		// 清空指定队列组的死信任务
		qg := workerCfg.GetQueueGroup(req.QueueName)
		if qg != nil {
			for priority := range qg.Priorities {
				fullQueue := workerCfg.FullQueueName(req.QueueName, priority)
				deleted, err := inspector.DeleteAllArchivedTasks(fullQueue)
				if err == nil {
					clearedQueues = append(clearedQueues, fullQueue)
					totalDeleted += deleted
				}
			}
		}
	} else {
		// 清空所有队列组的死信任务
		for _, qg := range workerCfg.QueueGroups {
			for priority := range qg.Priorities {
				fullQueue := workerCfg.FullQueueName(qg.Name, priority)
				deleted, err := inspector.DeleteAllArchivedTasks(fullQueue)
				if err != nil {
					continue
				}

				clearedQueues = append(clearedQueues, fullQueue)
				totalDeleted += deleted
			}
		}
	}

	c.JSON(http.StatusOK, dto.ClearDeadQueueResponse{
		Status:        "ok",
		WorkerName:    req.WorkerName,
		ClearedQueues: clearedQueues,
		TotalDeleted:  totalDeleted,
	})
}
