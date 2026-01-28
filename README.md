# TaskPM

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**一个通用的分布式任务管理和队列系统**

[English](README.md) | [中文文档](README_zh.md)

</div>

## ✨ 特性

- 🚀 **高性能**: 基于 Go 和 Asynq，支持高并发任务处理
- 📦 **开箱即用**: 提供简洁的 Worker SDK，快速集成å
- 🎯 **分布式**: 原生支持分布式部署和水平扩展
- 💪 **可靠性**: 任务失败自动重试，支持死信队列
- 📊 **可观测**: 内置 Prometheus 监控和 Web UI 管理界面
- 🔧 **易部署**: 单二进制部署，支持 Docker、Kubernetes
- 🌐 **Web UI**: 嵌入式 Web 界面，实时监控和管理
- 📖 **API 文档**: 完整的 Swagger API 文档

## 📋 目录

- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [使用指南](#使用指南)
- [API 文档](#api-文档)
- [部署方式](#部署方式)
- [性能指标](#性能指标)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

## 🚀 快速开始

### 前置要求

- Go 1.25+
- PostgreSQL 18+
- Redis 最新版
- pnpm (可选，用于前端开发)

### 安装

```bash
# 克隆仓库
git clone https://github.com/azhengyongqin/taskpm.git
cd taskpm

# 安装依赖
go mod download
```

### 本地开发

```bash
# 1. 启动依赖服务（PostgreSQL + Redis）
docker-compose up -d postgres redis

# 2. 运行数据库迁移
cd web && pnpm prisma:migrate:dev && cd ..

# 3. 构建并启动服务（包含嵌入的 Web UI）
make embed-web
make run

# 4. 启动 Worker 示例
make run-example
```

### 访问服务

- **Web UI**: <http://localhost:28080/>
- **API 文档**: <http://localhost:28080/swagger/index.html>
- **API 端点**: <http://localhost:28080/api/v1/>
- **健康检查**: <http://localhost:28080/healthz>
- **Prometheus**: <http://localhost:28080/metrics>
- **Asynqmon**: <http://localhost:8083>

## 📚 核心概念

### 架构概览

```
┌─────────────┐
│  业务系统    │
└──────┬──────┘
       │ 创建任务
       ↓
┌─────────────┐      ┌─────────────┐
│  TaskPM     │←────→│   Redis     │
│  Server     │      │   (队列)     │
└──────┬──────┘      └─────────────┘
       │
       ↓
┌─────────────┐      ┌─────────────┐
│  Worker     │←────→│ PostgreSQL  │
│  (SDK)      │      │  (持久化)    │
└─────────────┘      └─────────────┘
```

### 核心组件

1. **TaskPM Server**:
   - RESTful API 服务
   - 任务调度和管理
   - Worker 注册和心跳
   - 实时监控和统计

2. **Worker SDK**:
   - 简单易用的 Go SDK
   - 自动注册和心跳
   - 任务执行和重试
   - 状态上报

3. **Asynq 队列**:
   - 基于 Redis 的分布式队列
   - 支持优先级和延迟执行
   - 自动重试机制
   - 死信队列

4. **PostgreSQL**:
   - 任务元数据存储
   - 执行历史追踪
   - 统计数据查询

## 🔧 使用指南

### 创建任务

使用 HTTP API 创建任务：

```bash
curl -X POST http://localhost:28080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "worker_name": "my-worker",
    "queue": "default",
    "payload": {"url": "https://example.com"},
    "priority": 0,
    "delay_seconds": 0
  }'
```

### 实现 Worker

使用 SDK 快速实现 Worker：

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/azhengyongqin/taskpm/sdk"
)

func main() {
    // 创建 Worker 配置
    config := sdk.WorkerConfig{
        WorkerName:  "my-worker",
        BaseURL:     "http://localhost:28080",
        RedisAddr:   "redis://localhost:16379/0",
        Concurrency: 10,
        Queues: map[string]int{
            "my-worker:default": 10,
            "my-worker:high":    8,
        },
    }
    
    // 创建 Worker 实例
    worker := sdk.NewWorker(config)
    
    // 注册任务处理器
    worker.HandleFunc("default", func(ctx context.Context, payload json.RawMessage) error {
        // 解析 payload
        var task map[string]interface{}
        if err := json.Unmarshal(payload, &task); err != nil {
            return err
        }

        // 请求外部api
        resp, err := crawl.api()
        return err

        // Kafaf
        // 

        // Save 
        
        
        // 处理任务逻辑
        log.Printf("Processing task: %v", task)
        
        return nil
    })
    
    // 启动 Worker
    if err := worker.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### SDK 功能特性

```go
// 1. 多队列支持
worker.HandleFunc("default", defaultHandler)
worker.HandleFunc("high-priority", highPriorityHandler)
worker.HandleFunc("email", emailHandler)

// 2. 错误处理
worker.HandleFunc("default", func(ctx context.Context, payload json.RawMessage) error {
    // 返回 error 会自动重试
    if err := processTask(payload); err != nil {
        return fmt.Errorf("task failed: %w", err)
    }
    return nil
})

// 3. Context 支持
worker.HandleFunc("default", func(ctx context.Context, payload json.RawMessage) error {
    // 支持超时控制
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        return processTask(payload)
    }
})

// 4. 优雅关闭
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 监听系统信号
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    cancel()
}()

worker.Start(ctx)
```

## 📖 API 文档

完整的 API 文档通过 Swagger 提供：<http://localhost:28080/swagger/index.html>

### 主要端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/tasks` | POST | 创建任务 |
| `/api/v1/tasks` | GET | 查询任务列表 |
| `/api/v1/tasks/{id}` | GET | 获取任务详情 |
| `/api/v1/tasks/{id}/replay` | POST | 重放失败任务 |
| `/api/v1/tasks/batch-retry` | POST | 批量重试失败任务 |
| `/api/v1/workers` | GET | 获取 Worker 列表 |
| `/api/v1/workers/{name}/stats` | GET | Worker 统计信息 |
| `/api/v1/queues/stats` | GET | 队列统计信息 |
| `/api/v1/queues/clear` | POST | 清空指定队列 |
| `/api/v1/queues/clear-dead` | POST | 清空死信队列 |

## 🚢 部署方式

### Docker Compose

最简单的部署方式：

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f taskpm

# 停止服务
docker-compose down
```

访问：

- Web UI: <http://localhost:28080/>
- Asynqmon: <http://localhost:8083>

### Kubernetes

使用 Kustomize 部署：

```bash
# 开发环境
kubectl apply -k deployments/k8s/overlays/dev

# 生产环境
kubectl apply -k deployments/k8s/overlays/prod

# 查看部署状态
kubectl get pods -n taskpm
```

### Helm

使用 Helm Charts 部署：

```bash
# 安装
helm install taskpm deployments/helm/taskpm \
  -f deployments/helm/taskpm/values.yaml \
  --namespace taskpm \
  --create-namespace

# 升级
helm upgrade taskpm deployments/helm/taskpm \
  -f deployments/helm/taskpm/values.yaml

# 卸载
helm uninstall taskpm -n taskpm
```

### 二进制部署

```bash
# 构建
make build-all

# 运行服务端
./bin/server

# 运行 Worker
./bin/example
```

## 📊 性能指标

### 基准测试

| 指标 | 数值 |
|------|------|
| API 响应时间 (P95) | < 100ms |
| 任务吞吐量 | > 1000 tasks/s |
| 内存使用 | < 2GB |
| CPU 使用 | < 70% |
| 并发连接 | > 10000 |

### 可靠性指标

| 指标 | 数值 |
|------|------|
| 服务可用性 | > 99.9% |
| 任务成功率 | > 95% |
| 故障恢复时间 | < 30s |
| 数据一致性 | 100% |

## 🏗️ 项目结构

```
taskpm/
├── cmd/              # 可执行程序
│   ├── server/      # TaskPM 服务端
│   └── example/     # Worker 示例
├── sdk/             # Worker SDK
├── internal/        # 内部包
│   ├── server/      # HTTP 服务
│   ├── repository/  # 数据访问层
│   ├── queue/       # 队列管理
│   └── worker/      # Worker 管理
├── web/             # Web UI
├── deployments/     # 部署配置
│   ├── docker/      # Docker 配置
│   ├── k8s/         # Kubernetes 配置
│   └── helm/        # Helm Charts
├── docs/            # 文档
└── prisma/          # 数据库 Schema
```

详细架构说明请查看 [架构文档](docs/ARCHITECTURE.md)

## 🛠️ 开发指南

### 构建命令

```bash
# 编译所有服务
make build-all

# 构建前端并嵌入到 Go 服务
make embed-web

# 生成 Swagger 文档
make swagger

# 运行测试
make test

# 代码检查
make lint

# 构建 Docker 镜像
make docker-build
```

### Web UI

Web UI 已通过 Go embed 嵌入到服务端二进制中，无需单独部署。

**本地开发：**

```bash
cd web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建生产版本
pnpm build
```

### 数据库迁移

```bash
# 创建迁移
cd web && pnpm prisma migrate dev --name migration_name

# 应用迁移
pnpm prisma migrate deploy

# 查看迁移状态
pnpm prisma migrate status
```

## 🌟 功能特性

### ✅ 已实现

- [x] 分布式任务调度
- [x] 多队列优先级支持
- [x] 任务失败自动重试
- [x] Worker 自动注册和心跳
- [x] 实时监控和统计
- [x] Web UI 管理界面
- [x] RESTful API
- [x] Swagger API 文档
- [x] Prometheus 监控
- [x] 健康检查
- [x] 批量操作
- [x] Docker 部署
- [x] Kubernetes 部署
- [x] Helm Charts

### 🚧 计划中

- [ ] gRPC 支持
- [ ] 任务依赖关系
- [ ] 定时任务 (Cron)
- [ ] 工作流编排
- [ ] 多租户支持
- [ ] OpenTelemetry 集成
- [ ] 分布式追踪
- [ ] 更多语言的 SDK (Python, Node.js, Java)

## 🤝 贡献指南

我们欢迎各种形式的贡献！请阅读 [贡献指南](CONTRIBUTING.md) 了解如何参与项目。

### 贡献者

感谢所有为 TaskPM 做出贡献的开发者！

<!-- ALL-CONTRIBUTORS-LIST:START -->
<!-- 贡献者列表将自动更新 -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

## 📄 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## 🔗 相关链接

- [架构文档](docs/ARCHITECTURE.md)
- [API 文档](http://localhost:28080/swagger/index.html)
- [贡献指南](CONTRIBUTING.md)
- [更新日志](CHANGELOG.md)
- [问题反馈](https://github.com/azhengyongqin/taskpm/issues)

## 📞 联系我们

- 提交 Issue: [GitHub Issues](https://github.com/azhengyongqin/taskpm/issues)
- 讨论交流: [GitHub Discussions](https://github.com/azhengyongqin/taskpm/discussions)

## ⭐ Star History

如果这个项目对您有帮助，请给我们一个 Star！

[![Star History Chart](https://api.star-history.com/svg?repos=azhengyongqin/taskpm&type=Date)](https://star-history.com/#azhengyongqin/taskpm&Date)

---

<div align="center">
Made with ❤️ by TaskPM Team
</div>
