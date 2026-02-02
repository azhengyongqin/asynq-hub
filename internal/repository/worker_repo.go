package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	workers "github.com/azhengyongqin/asynq-hub/internal/worker"
)

// WorkerRepo Worker 仓储实现
type WorkerRepo struct {
	db *gorm.DB
}

// NewWorkerRepo 创建 Worker 仓储
func NewWorkerRepo(db *gorm.DB) *WorkerRepo {
	return &WorkerRepo{db: db}
}

// Upsert 插入或更新 worker 配置
func (r *WorkerRepo) Upsert(ctx context.Context, c WorkerConfig) error {
	model := WorkerConfigToModel(c)
	model.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "worker_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"base_url", "redis_addr", "queue_groups",
			"default_retry_count", "default_timeout", "default_delay",
			"is_enabled", "last_heartbeat_at", "updated_at",
		}),
	}).Create(&model).Error
}

// List 列出所有 worker
func (r *WorkerRepo) List(ctx context.Context) ([]WorkerConfig, error) {
	var models []WorkerModel
	if err := r.db.WithContext(ctx).Order("worker_name ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	configs := make([]WorkerConfig, len(models))
	for i, m := range models {
		configs[i] = m.ToWorkerConfig()
	}
	return configs, nil
}

// Get 获取单个 worker
func (r *WorkerRepo) Get(ctx context.Context, workerName string) (*WorkerConfig, error) {
	var model WorkerModel
	if err := r.db.WithContext(ctx).Where("worker_name = ?", workerName).First(&model).Error; err != nil {
		return nil, err
	}
	config := model.ToWorkerConfig()
	return &config, nil
}

// UpdateHeartbeat 更新心跳时间
func (r *WorkerRepo) UpdateHeartbeat(ctx context.Context, workerName string, heartbeatAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&WorkerModel{}).
		Where("worker_name = ?", workerName).
		Updates(map[string]interface{}{
			"last_heartbeat_at": heartbeatAt,
			"updated_at":        time.Now(),
		}).Error
}

// Delete 删除 worker
func (r *WorkerRepo) Delete(ctx context.Context, workerName string) error {
	return r.db.WithContext(ctx).
		Where("worker_name = ?", workerName).
		Delete(&WorkerModel{}).Error
}

// ListOfflineWorkers 查询离线的 Worker 列表（心跳超过指定时间）
func (r *WorkerRepo) ListOfflineWorkers(ctx context.Context, offlineDuration time.Duration) ([]WorkerConfig, error) {
	cutoffTime := time.Now().Add(-offlineDuration)

	var models []WorkerModel
	if err := r.db.WithContext(ctx).
		Where("last_heartbeat_at IS NULL OR last_heartbeat_at < ?", cutoffTime).
		Order("worker_name ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	configs := make([]WorkerConfig, len(models))
	for i, m := range models {
		configs[i] = m.ToWorkerConfig()
	}
	return configs, nil
}

// EnsureSeed 种子数据（开发环境使用）
func (r *WorkerRepo) EnsureSeed(ctx context.Context, seed []workers.Config) error {
	// 检查是否已有数据
	var count int64
	if err := r.db.WithContext(ctx).Model(&WorkerModel{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// 插入种子数据
	for _, c := range seed {
		// 转换队列组配置
		queueGroups := make([]QueueGroupConfig, len(c.QueueGroups))
		for i, qg := range c.QueueGroups {
			queueGroups[i] = QueueGroupConfig{
				Name:        qg.Name,
				Concurrency: qg.Concurrency,
				Priorities:  qg.Priorities,
			}
		}

		repoConfig := WorkerConfig{
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
		if err := r.Upsert(ctx, repoConfig); err != nil {
			return err
		}
	}

	// 防止读到旧快照
	time.Sleep(10 * time.Millisecond)
	return nil
}
