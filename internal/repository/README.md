# Asynq-Hub 持久化层抽象

## 概述

为了支持未来迁移到 ClickHouse，我们对持久化层进行了抽象，定义了 `TaskRepository` 和 `WorkerRepository` 接口。

## 接口定义

### TaskRepository 接口

位置：`backend/internal/repository/repository.go`

主要方法：
- `UpsertTask` - 创建或更新任务
- `UpdateTaskStatus` - 更新任务状态  
- `GetTask` - 获取任务详情
- `ListTasks` - 查询任务列表（支持分页和过滤）
- `CountTasks` - 统计任务总数
- `InsertAttempt` - 插入任务执行尝试记录
- `ListAttempts` - 查询任务的执行尝试历史
- `GetWorkerStats` - 获取 Worker 统计信息
- `GetWorkerTimeSeriesStats` - 获取 Worker 时间序列统计数据
- `ListFailedTasks` - 查询失败的任务列表（用于批量重试）

### WorkerRepository 接口

位置：`backend/internal/repository/worker_repository.go`

主要方法：
- `Upsert` - 创建或更新 Worker 配置
- `Get` - 获取 Worker 配置
- `List` - 查询所有 Worker 配置列表
- `Delete` - 删除 Worker 配置
- `UpdateHeartbeat` - 更新 Worker 心跳时间
- `ListOfflineWorkers` - 查询离线的 Worker 列表

## 实现

### PostgreSQL 实现

当前实现位于：
- `backend/internal/repository/postgresrepo/task_repo.go`
- `backend/internal/repository/postgresrepo/worker_repo.go`

PostgreSQL 实现返回接口类型，确保与接口定义兼容。

### ClickHouse 实现（待实现）

未来可以在 `backend/internal/repository/clickhouserepo/` 目录下实现相同接口，用于支持 ClickHouse 作为数据存储。

## 使用方式

在 HTTP 层，依赖接口而非具体实现：

```go
type Deps struct {
    WorkerStore *workers.Store
    AsynqClient *asynq.Client
    WorkerRepo  repository.WorkerRepository  // 接口类型
    TaskRepo    repository.TaskRepository    // 接口类型
}
```

在 `main.go` 中注入具体实现：

```go
// 使用 PostgreSQL 实现
taskRepo := postgresrepo.NewTaskRepo(pgPool.Pool)
workerRepo := postgresrepo.NewWorkerRepo(pgPool.Pool)

// 注入到 HTTP 层
httpSrv := &http.Server{
    Addr:    httpAddr,
    Handler: httpserver.NewRouter(httpserver.Deps{
        WorkerStore: workerStore,
        AsynqClient: asynqClient,
        WorkerRepo:  workerRepo,  // 接口类型
        TaskRepo:    taskRepo,     // 接口类型
    }),
    ReadHeaderTimeout: 5 * time.Second,
}
```

## 迁移到 ClickHouse

当需要迁移到 ClickHouse 时，只需：

1. 实现 ClickHouse 版本的 Repository
2. 修改 main.go 中的注入逻辑
3. HTTP 层代码无需修改

示例：

```go
// 使用 ClickHouse 实现
taskRepo := clickhouserepo.NewTaskRepo(chConn)
workerRepo := clickhouserepo.NewWorkerRepo(chConn)

// 其他代码不变
```

## 优势

1. **解耦**：HTTP 层不依赖具体的数据库实现
2. **可测试**：可以轻松创建 Mock 实现用于测试
3. **灵活性**：支持多种数据库实现共存
4. **迁移成本低**：迁移数据库时只需替换实现，业务逻辑无需改动

## 注意事项

- 接口中的类型定义（Task, Attempt, WorkerConfig 等）位于 `repository` 包
- PostgreSQL 实现需要与接口定义的类型保持一致
- 新增方法时需要同步更新接口和所有实现
