package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azhengyongqin/taskpm/internal/worker"
)

type WorkerRepo struct {
	pool *pgxpool.Pool
}

func NewWorkerRepo(pool *pgxpool.Pool) *WorkerRepo {
	return &WorkerRepo{pool: pool}
}

// Upsert 插入或更新 worker 配置
func (r *WorkerRepo) Upsert(ctx context.Context, c workers.Config) error {
	queuesJSON, _ := json.Marshal(c.Queues)

	_, err := r.pool.Exec(ctx, `
insert into worker(worker_name, base_url, redis_addr, concurrency, queues, default_retry_count, default_timeout, default_delay, is_enabled, last_heartbeat_at)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
on conflict (worker_name) do update
set base_url=excluded.base_url,
    redis_addr=excluded.redis_addr,
    concurrency=excluded.concurrency,
    queues=excluded.queues,
    default_retry_count=excluded.default_retry_count,
    default_timeout=excluded.default_timeout,
    default_delay=excluded.default_delay,
    is_enabled=excluded.is_enabled,
    last_heartbeat_at=excluded.last_heartbeat_at,
    updated_at=now()
`, c.WorkerName, c.BaseURL, c.RedisAddr, c.Concurrency, queuesJSON, c.DefaultRetryCount, c.DefaultTimeout, c.DefaultDelay, c.IsEnabled, c.LastHeartbeatAt)
	return err
}

// List 列出所有 worker
func (r *WorkerRepo) List(ctx context.Context) ([]workers.Config, error) {
	rows, err := r.pool.Query(ctx, `
select worker_name, coalesce(base_url,''), coalesce(redis_addr,''), concurrency, queues, default_retry_count, default_timeout, default_delay, is_enabled, last_heartbeat_at
from worker
order by worker_name asc
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []workers.Config
	for rows.Next() {
		var c workers.Config
		var queuesJSON []byte
		if err := rows.Scan(&c.WorkerName, &c.BaseURL, &c.RedisAddr, &c.Concurrency, &queuesJSON, &c.DefaultRetryCount, &c.DefaultTimeout, &c.DefaultDelay, &c.IsEnabled, &c.LastHeartbeatAt); err != nil {
			return nil, err
		}
		// 解析 queues JSON
		if err := json.Unmarshal(queuesJSON, &c.Queues); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get 获取单个 worker
func (r *WorkerRepo) Get(ctx context.Context, workerName string) (*workers.Config, error) {
	row := r.pool.QueryRow(ctx, `
select worker_name, coalesce(base_url,''), coalesce(redis_addr,''), concurrency, queues, default_retry_count, default_timeout, default_delay, is_enabled, last_heartbeat_at
from worker
where worker_name=$1
`, workerName)

	var c workers.Config
	var queuesJSON []byte
	if err := row.Scan(&c.WorkerName, &c.BaseURL, &c.RedisAddr, &c.Concurrency, &queuesJSON, &c.DefaultRetryCount, &c.DefaultTimeout, &c.DefaultDelay, &c.IsEnabled, &c.LastHeartbeatAt); err != nil {
		return nil, err
	}
	// 解析 queues JSON
	if err := json.Unmarshal(queuesJSON, &c.Queues); err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateHeartbeat 更新心跳时间
func (r *WorkerRepo) UpdateHeartbeat(ctx context.Context, workerName string, heartbeatAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
update worker
set last_heartbeat_at=$2, updated_at=now()
where worker_name=$1
`, workerName, heartbeatAt)
	return err
}

// Delete 删除 worker
func (r *WorkerRepo) Delete(ctx context.Context, workerName string) error {
	_, err := r.pool.Exec(ctx, `
delete from worker where worker_name=$1
`, workerName)
	return err
}

// EnsureSeed 种子数据（开发环境使用）
func (r *WorkerRepo) EnsureSeed(ctx context.Context, seed []workers.Config) error {
	// 检查是否已有数据
	row := r.pool.QueryRow(ctx, `select count(1) from worker`)
	var n int64
	if err := row.Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	// 插入种子数据
	for _, c := range seed {
		if err := r.Upsert(ctx, c); err != nil {
			return err
		}
	}

	// 防止读到旧快照
	time.Sleep(10 * time.Millisecond)
	return nil
}
