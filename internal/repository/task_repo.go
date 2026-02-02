package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TaskRepo 任务仓储实现
type TaskRepo struct {
	db *gorm.DB
}

// NewTaskRepo 创建任务仓储
func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// UpsertTask 创建或更新任务
func (r *TaskRepo) UpsertTask(ctx context.Context, t Task) error {
	if t.TaskID == "" {
		return errors.New("task_id 不能为空")
	}

	model := TaskToModel(t)
	model.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "task_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"queue", "priority", "payload", "status",
			"last_attempt", "last_error", "last_worker_name", "trace_id", "updated_at",
		}),
	}).Create(&model).Error
}

// UpdateTaskStatus 更新任务状态
func (r *TaskRepo) UpdateTaskStatus(ctx context.Context, taskID, status string, lastAttempt int, lastError string, lastWorkerName string) error {
	updates := map[string]interface{}{
		"status":           status,
		"last_attempt":     lastAttempt,
		"last_error":       lastError,
		"last_worker_name": lastWorkerName,
		"updated_at":       time.Now(),
	}

	return r.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("task_id = ?", taskID).
		Updates(updates).Error
}

// GetTask 获取任务详情
func (r *TaskRepo) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var model TaskModel
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&model).Error; err != nil {
		return nil, err
	}
	task := model.ToTask()
	return &task, nil
}

// ListTasks 查询任务列表
func (r *TaskRepo) ListTasks(ctx context.Context, f ListTasksFilter) ([]Task, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := r.db.WithContext(ctx).Model(&TaskModel{})

	if f.WorkerName != "" {
		query = query.Where("worker_name = ?", f.WorkerName)
	}
	if f.Status != "" {
		query = query.Where("status = ?", f.Status)
	}
	if f.Queue != "" {
		query = query.Where("queue = ?", f.Queue)
	}

	var models []TaskModel
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	tasks := make([]Task, len(models))
	for i, m := range models {
		tasks[i] = m.ToTask()
	}
	return tasks, nil
}

// CountTasks 统计任务数量
func (r *TaskRepo) CountTasks(ctx context.Context, f ListTasksFilter) (int, error) {
	query := r.db.WithContext(ctx).Model(&TaskModel{})

	if f.WorkerName != "" {
		query = query.Where("worker_name = ?", f.WorkerName)
	}
	if f.Status != "" {
		query = query.Where("status = ?", f.Status)
	}
	if f.Queue != "" {
		query = query.Where("queue = ?", f.Queue)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// InsertAttempt 插入任务执行尝试记录
func (r *TaskRepo) InsertAttempt(ctx context.Context, a Attempt) error {
	model := AttemptToModel(a)
	return r.db.WithContext(ctx).Create(&model).Error
}

// ListAttempts 查询任务执行尝试历史
func (r *TaskRepo) ListAttempts(ctx context.Context, taskID string, limit int) ([]Attempt, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var models []TaskAttemptModel
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("started_at DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}

	attempts := make([]Attempt, len(models))
	for i, m := range models {
		attempts[i] = m.ToAttempt()
	}
	return attempts, nil
}

// GetWorkerStats 获取指定 worker 的统计信息
func (r *TaskRepo) GetWorkerStats(ctx context.Context, workerName string) (*WorkerStats, error) {
	stats := &WorkerStats{
		QueueStats:        make(map[string]int),
		QueueSuccessStats: make(map[string]int),
		QueueFailedStats:  make(map[string]int),
		QueueAvgDuration:  make(map[string]int),
	}

	// 统计各状态任务数量
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount

	if err := r.db.WithContext(ctx).
		Model(&TaskModel{}).
		Select("status, count(*) as count").
		Where("worker_name = ?", workerName).
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}

	for _, sc := range statusCounts {
		stats.TotalTasks += sc.Count
		switch sc.Status {
		case "pending":
			stats.PendingTasks = sc.Count
		case "running":
			stats.RunningTasks = sc.Count
		case "success":
			stats.SuccessTasks = sc.Count
		case "fail":
			stats.FailedTasks = sc.Count
		}
	}

	// 计算成功率
	if stats.TotalTasks > 0 {
		stats.SuccessRate = float64(stats.SuccessTasks) / float64(stats.TotalTasks) * 100
	}

	// 计算平均执行时间
	var avgDuration *int
	r.db.WithContext(ctx).
		Model(&TaskAttemptModel{}).
		Select("COALESCE(AVG(duration_ms)::int, 0)").
		Joins("JOIN task ON task_attempt.task_id = task.task_id").
		Where("task.worker_name = ? AND task_attempt.duration_ms IS NOT NULL AND task_attempt.status = 'success'", workerName).
		Scan(&avgDuration)

	if avgDuration != nil {
		stats.AvgDurationMs = *avgDuration
	}

	// 按队列统计
	type QueueCount struct {
		Queue string
		Total int
	}
	var queueCounts []QueueCount

	if err := r.db.WithContext(ctx).
		Model(&TaskModel{}).
		Select("queue, count(*) as total").
		Where("worker_name = ?", workerName).
		Group("queue").
		Scan(&queueCounts).Error; err == nil {
		for _, qc := range queueCounts {
			stats.QueueStats[qc.Queue] = qc.Total
		}
	}

	return stats, nil
}

// GetWorkerTimeSeriesStats 获取 worker 的时间序列统计
func (r *TaskRepo) GetWorkerTimeSeriesStats(ctx context.Context, workerName string, hours int) ([]TimeSeriesStats, error) {
	if hours <= 0 || hours > 168 {
		hours = 24
	}

	var results []TimeSeriesStats

	// 使用原生 SQL 进行时间序列聚合
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			to_char(date_trunc('hour', ta.started_at), 'YYYY-MM-DD HH24:00') as hour,
			count(*) as total_tasks,
			sum(case when ta.status = 'success' then 1 else 0 end)::int as success_tasks,
			sum(case when ta.status in ('fail', 'dead') then 1 else 0 end)::int as failed_tasks,
			coalesce(avg(case when ta.status = 'success' and ta.duration_ms is not null then ta.duration_ms else null end)::int, 0) as avg_duration
		FROM task_attempt ta
		JOIN task t ON ta.task_id = t.task_id
		WHERE t.worker_name = ? 
		  AND ta.started_at >= now() - interval '1 hour' * ?
		GROUP BY hour
		ORDER BY hour ASC
	`, workerName, hours).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// ListFailedTasks 查询失败的任务列表（用于批量重试）
func (r *TaskRepo) ListFailedTasks(ctx context.Context, workerName string, limit int) ([]Task, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	query := r.db.WithContext(ctx).Model(&TaskModel{}).Where("status = 'fail'")

	if workerName != "" {
		query = query.Where("worker_name = ?", workerName)
	}

	var models []TaskModel
	if err := query.Order("created_at DESC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	tasks := make([]Task, len(models))
	for i, m := range models {
		tasks[i] = m.ToTask()
	}
	return tasks, nil
}
