package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) UpsertTask(ctx context.Context, t Task) error {
	if t.TaskID == "" {
		return errors.New("task_id 不能为空")
	}
	_, err := r.pool.Exec(ctx, `
insert into task(task_id, worker_name, queue, priority, payload, status, last_attempt, last_error, last_worker_name, trace_id)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
on conflict (task_id) do update
set queue = excluded.queue,
    priority = excluded.priority,
    payload = excluded.payload,
    status = excluded.status,
    last_attempt = excluded.last_attempt,
    last_error = excluded.last_error,
    last_worker_name = excluded.last_worker_name,
    trace_id = excluded.trace_id,
    updated_at = now()
`, t.TaskID, t.WorkerName, t.Queue, t.Priority, t.Payload, t.Status, t.LastAttempt, t.LastError, t.LastWorkerName, t.TraceID)
	return err
}

func (r *TaskRepo) UpdateTaskStatus(ctx context.Context, taskID, status string, lastAttempt int, lastError string, lastWorkerName string) error {
	_, err := r.pool.Exec(ctx, `
update task
set status=$2, last_attempt=$3, last_error=$4, last_worker_name=$5, updated_at=now()
where task_id=$1
`, taskID, status, lastAttempt, lastError, lastWorkerName)
	return err
}

func (r *TaskRepo) GetTask(ctx context.Context, taskID string) (*Task, error) {
	row := r.pool.QueryRow(ctx, `
select task_id, worker_name, queue, priority, payload, status, last_attempt, coalesce(last_error,''), coalesce(last_worker_name,''), coalesce(trace_id,''), created_at, updated_at
from task
where task_id=$1
`, taskID)

	var t Task
	if err := row.Scan(&t.TaskID, &t.WorkerName, &t.Queue, &t.Priority, &t.Payload, &t.Status, &t.LastAttempt, &t.LastError, &t.LastWorkerName, &t.TraceID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepo) ListTasks(ctx context.Context, f ListTasksFilter) ([]Task, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, `
select task_id, worker_name, queue, priority, payload, status, last_attempt, coalesce(last_error,''), coalesce(last_worker_name,''), coalesce(trace_id,''), created_at, updated_at
from task
where ($1='' or worker_name=$1)
  and ($2='' or status=$2)
  and ($3='' or queue=$3)
order by created_at desc
limit $4 offset $5
`, f.WorkerName, f.Status, f.Queue, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.TaskID, &t.WorkerName, &t.Queue, &t.Priority, &t.Payload, &t.Status, &t.LastAttempt, &t.LastError, &t.LastWorkerName, &t.TraceID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TaskRepo) CountTasks(ctx context.Context, f ListTasksFilter) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
select count(*)
from task
where ($1='' or worker_name=$1)
  and ($2='' or status=$2)
  and ($3='' or queue=$3)
`, f.WorkerName, f.Status, f.Queue).Scan(&count)
	return count, err
}

func (r *TaskRepo) InsertAttempt(ctx context.Context, a Attempt) error {
	_, err := r.pool.Exec(ctx, `
insert into task_attempt(task_id, asynq_task_id, attempt, status, started_at, finished_at, duration_ms, error, worker_name, trace_id, span_id)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
`, a.TaskID, a.AsynqTaskID, a.Attempt, a.Status, a.StartedAt, a.FinishedAt, a.DurationMs, a.Error, a.WorkerName, a.TraceID, a.SpanID)
	return err
}

func (r *TaskRepo) ListAttempts(ctx context.Context, taskID string, limit int) ([]Attempt, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
select task_id, coalesce(asynq_task_id,''), attempt, status, started_at, finished_at, duration_ms, coalesce(error,''), coalesce(worker_name,''), coalesce(trace_id,''), coalesce(span_id,'')
from task_attempt
where task_id=$1
order by started_at desc
limit $2
`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Attempt
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.TaskID, &a.AsynqTaskID, &a.Attempt, &a.Status, &a.StartedAt, &a.FinishedAt, &a.DurationMs, &a.Error, &a.WorkerName, &a.TraceID, &a.SpanID); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetWorkerStats 获取指定 worker 的统计信息

// GetWorkerStats 获取指定 worker 的统计信息
func (r *TaskRepo) GetWorkerStats(ctx context.Context, workerName string) (*WorkerStats, error) {
	stats := &WorkerStats{
		QueueStats:        make(map[string]int),
		QueueSuccessStats: make(map[string]int),
		QueueFailedStats:  make(map[string]int),
		QueueAvgDuration:  make(map[string]int),
	}

	// 统计各状态任务数量
	rows, err := r.pool.Query(ctx, `
select status, count(*) as cnt
from task
where worker_name = $1
group by status
`, workerName)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return nil, err
		}

		stats.TotalTasks += count

		switch status {
		case "pending":
			stats.PendingTasks = count
		case "running":
			stats.RunningTasks = count
		case "success":
			stats.SuccessTasks = count
		case "fail":
			stats.FailedTasks = count
		}
	}
	rows.Close()

	// 计算成功率
	if stats.TotalTasks > 0 {
		stats.SuccessRate = float64(stats.SuccessTasks) / float64(stats.TotalTasks) * 100
	}

	// 计算平均执行时间
	err = r.pool.QueryRow(ctx, `
select coalesce(avg(duration_ms)::int, 0)
from task_attempt
where task_id in (select task_id from task where worker_name = $1) 
  and duration_ms is not null 
  and status = 'success'
`, workerName).Scan(&stats.AvgDurationMs)
	if err != nil {
		stats.AvgDurationMs = 0
	}

	// 按队列统计
	queueRows, err := r.pool.Query(ctx, `
select queue, count(*) as total
from task
where worker_name = $1
group by queue
`, workerName)
	if err == nil {
		for queueRows.Next() {
			var queue string
			var total int
			if err := queueRows.Scan(&queue, &total); err == nil {
				stats.QueueStats[queue] = total
			}
		}
		queueRows.Close()
	}

	return stats, nil
}

// GetWorkerTimeSeriesStats 获取 worker 的时间序列统计
func (r *TaskRepo) GetWorkerTimeSeriesStats(ctx context.Context, workerName string, hours int) ([]TimeSeriesStats, error) {
	if hours <= 0 || hours > 168 {
		hours = 24
	}

	rows, err := r.pool.Query(ctx, `
select 
    to_char(date_trunc('hour', ta.started_at), 'YYYY-MM-DD HH24:00') as hour,
    count(*) as total,
    sum(case when ta.status = 'success' then 1 else 0 end)::int as success_count,
    sum(case when ta.status in ('fail', 'dead') then 1 else 0 end)::int as fail_count,
    coalesce(avg(case when ta.status = 'success' and ta.duration_ms is not null then ta.duration_ms else null end)::int, 0) as avg_duration
from task_attempt ta
join task t on ta.task_id = t.task_id
where t.worker_name = $1 
  and ta.started_at >= now() - interval '1 hour' * $2
group by hour
order by hour asc
`, workerName, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimeSeriesStats
	for rows.Next() {
		var ts TimeSeriesStats
		if err := rows.Scan(&ts.Hour, &ts.TotalTasks, &ts.SuccessTasks, &ts.FailedTasks, &ts.AvgDuration); err != nil {
			return nil, err
		}
		result = append(result, ts)
	}

	return result, rows.Err()
}

// ListFailedTasks 查询失败的任务列表（用于批量重试）
func (r *TaskRepo) ListFailedTasks(ctx context.Context, workerName string, limit int) ([]Task, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx, `
select task_id, worker_name, queue, priority, payload, status, last_attempt, coalesce(last_error,''), coalesce(last_worker_name,''), coalesce(trace_id,''), created_at, updated_at
from task
where status = 'fail'
  and ($1='' or worker_name=$1)
order by created_at desc
limit $2
`, workerName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.TaskID, &t.WorkerName, &t.Queue, &t.Priority, &t.Payload, &t.Status, &t.LastAttempt, &t.LastError, &t.LastWorkerName, &t.TraceID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
