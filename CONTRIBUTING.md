# 贡献指南

感谢您对 Asynq-Hub 项目的关注！我们欢迎并感激所有形式的贡献。

## 📋 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [问题反馈](#问题反馈)
- [文档贡献](#文档贡献)

## 🤝 行为准则

### 我们的承诺

为了营造一个开放和友好的环境，我们承诺：

- 尊重不同的观点和经验
- 接受建设性的批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

### 不可接受的行为

- 使用性暗示的语言或图像
- 侮辱/贬损性评论和人身攻击
- 公开或私下的骚扰
- 未经许可发布他人的私人信息
- 其他不道德或不专业的行为

## 💡 如何贡献

### 贡献方式

您可以通过以下方式为项目做出贡献：

1. **报告 Bug**
   - 使用清晰的标题和描述
   - 提供复现步骤
   - 附上相关的日志和截图
   - 说明您的环境信息

2. **建议新功能**
   - 描述功能的使用场景
   - 说明为什么需要这个功能
   - 提供可能的实现方案

3. **提交代码**
   - 修复 Bug
   - 实现新功能
   - 改进性能
   - 完善文档

4. **改进文档**
   - 修正文档错误
   - 补充缺失的文档
   - 翻译文档
   - 添加示例代码

5. **帮助其他人**
   - 回答 Issue 中的问题
   - 审查 Pull Request
   - 分享使用经验

## 🛠️ 开发环境搭建

### 前置要求

确保您的系统已安装以下工具：

```bash
# 必需工具
- Go 1.25+
- PostgreSQL 18+
- Redis 最新版
- Git 2.0+

# 推荐工具
- Docker & Docker Compose
- Make
- pnpm (前端开发)
- golangci-lint (代码检查)
```

### 克隆仓库

```bash
# 1. Fork 项目到您的 GitHub 账号

# 2. 克隆您 fork 的仓库
git clone https://github.com/YOUR_USERNAME/asynq-hub.git
cd asynq-hub

# 3. 添加上游仓库
git remote add upstream https://github.com/azhengyongqin/asynq-hub.git

# 4. 验证远程仓库
git remote -v
```

### 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装前端依赖（可选）
cd web && pnpm install && cd ..

# 安装开发工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

### 启动开发环境

```bash
# 1. 启动依赖服务
docker-compose up -d postgres redis

# 2. 运行数据库迁移
cd web && pnpm prisma:migrate:dev && cd ..

# 3. 启动后端服务
make run

# 4. 启动前端（可选，用于开发）
cd web && pnpm dev

# 5. 启动 Worker 示例
make run-example
```

### 验证环境

```bash
# 检查后端
curl http://localhost:28080/healthz

# 检查 API
curl http://localhost:28080/api/v1/workers

# 访问 Web UI
open http://localhost:28080/

# 访问 Swagger
open http://localhost:28080/swagger/index.html
```

## 📝 代码规范

### Go 代码规范

遵循 [Effective Go](https://golang.org/doc/effective_go.html) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)。

#### 命名规范

```go
// ✅ 好的命名
type WorkerConfig struct {
    WorkerName  string
    BaseURL     string
    Concurrency int
}

func NewWorker(cfg WorkerConfig) *Worker {
    return &Worker{config: cfg}
}

// ❌ 不好的命名
type workerconfig struct {
    worker_name string
    baseUrl     string
    c           int
}

func newworker(c workerconfig) *Worker {
    return &Worker{cfg: c}
}
```

#### 注释规范

```go
// ✅ 导出的类型和函数必须有注释
// Worker 管理任务的执行和调度
// 它自动注册到 Asynq-Hub Server 并定期发送心跳
type Worker struct {
    config WorkerConfig
    // ... 字段
}

// NewWorker 创建一个新的 Worker 实例
// 参数:
//   - config: Worker 配置
// 返回:
//   - *Worker: Worker 实例
func NewWorker(config WorkerConfig) *Worker {
    return &Worker{config: config}
}

// ❌ 缺少注释
type Worker struct {
    config WorkerConfig
}

func NewWorker(config WorkerConfig) *Worker {
    return &Worker{config: config}
}
```

#### 错误处理

```go
// ✅ 明确的错误处理
func (w *Worker) Start(ctx context.Context) error {
    if err := w.register(); err != nil {
        return fmt.Errorf("failed to register worker: %w", err)
    }
    
    if err := w.startHeartbeat(); err != nil {
        return fmt.Errorf("failed to start heartbeat: %w", err)
    }
    
    return nil
}

// ❌ 忽略错误
func (w *Worker) Start(ctx context.Context) error {
    w.register() // 忽略错误
    w.startHeartbeat() // 忽略错误
    return nil
}
```

#### 包导入顺序

```go
import (
    // 1. 标准库
    "context"
    "encoding/json"
    "fmt"
    
    // 2. 第三方库
    "github.com/gin-gonic/gin"
    "github.com/hibiken/asynq"
    
    // 3. 项目内部包
    "github.com/azhengyongqin/asynq-hub/internal/config"
    "github.com/azhengyongqin/asynq-hub/internal/repository"
)
```

### TypeScript/React 规范

遵循 [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript) 和 [React 官方文档](https://react.dev/)。

```typescript
// ✅ 使用 TypeScript 类型
interface WorkerStats {
  totalTasks: number
  successRate: number
  avgDurationMs: number | null
}

function WorkerCard({ stats }: { stats: WorkerStats }) {
  return (
    <div>
      <h3>Total: {stats.totalTasks}</h3>
      <p>Success Rate: {stats.successRate.toFixed(1)}%</p>
    </div>
  )
}

// ❌ 缺少类型
function WorkerCard({ stats }) {
  return (
    <div>
      <h3>Total: {stats.totalTasks}</h3>
    </div>
  )
}
```

### 代码检查

```bash
# Go 代码检查
make lint
# 或
golangci-lint run ./...

# Go 代码格式化
go fmt ./...
gofmt -s -w .

# 前端代码检查（如果有）
cd web && pnpm lint

# 前端代码格式化
cd web && pnpm format
```

## 📌 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档变更
- `style`: 代码格式（不影响代码运行）
- `refactor`: 重构（既不是新功能也不是 Bug 修复）
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动
- `ci`: CI/CD 相关
- `revert`: 回滚提交

### 提交示例

```bash
# 好的提交
git commit -m "feat(sdk): add timeout configuration for worker"
git commit -m "fix(api): resolve task creation race condition"
git commit -m "docs(readme): update installation instructions"
git commit -m "perf(queue): optimize task polling performance"

# 不好的提交
git commit -m "update"
git commit -m "fix bug"
git commit -m "changes"
```

### 详细示例

```bash
feat(worker): add support for custom retry strategies

- Add RetryStrategy interface
- Implement ExponentialBackoff strategy
- Update Worker config to accept custom strategy
- Add unit tests for retry logic

Closes #123
```

## 🔀 Pull Request 流程

### 1. 准备工作

```bash
# 确保您的 fork 是最新的
git checkout main
git fetch upstream
git merge upstream/main
git push origin main

# 创建新分支
git checkout -b feature/my-awesome-feature
```

### 2. 开发

```bash
# 进行您的修改
# ...编写代码...

# 运行测试
make test

# 代码检查
make lint

# 提交代码
git add .
git commit -m "feat(scope): description"
```

### 3. 推送

```bash
# 推送到您的 fork
git push origin feature/my-awesome-feature
```

### 4. 创建 Pull Request

1. 访问 GitHub 上您 fork 的仓库
2. 点击 "New Pull Request"
3. 选择 base: `main` ← compare: `feature/my-awesome-feature`
4. 填写 PR 标题和描述

### PR 标题格式

```
feat(sdk): add retry configuration
fix(api): resolve race condition in task creation
docs: update contributing guidelines
```

### PR 描述模板

```markdown
## 变更说明

简要描述您的改动

## 变更类型

- [ ] Bug 修复
- [ ] 新功能
- [ ] 破坏性变更
- [ ] 文档更新
- [ ] 性能优化
- [ ] 其他（请说明）

## 测试

描述您如何测试这些改动

- [ ] 单元测试
- [ ] 集成测试
- [ ] 手动测试

## 检查清单

- [ ] 代码遵循项目规范
- [ ] 已添加必要的注释
- [ ] 已添加/更新测试
- [ ] 所有测试通过
- [ ] 已更新相关文档
- [ ] 提交信息遵循规范

## 相关 Issue

Closes #123
```

### 5. Code Review

- 响应审查者的评论
- 根据反馈进行修改
- 保持提交历史清晰

```bash
# 修改后继续提交
git add .
git commit -m "refactor: address review comments"
git push origin feature/my-awesome-feature
```

### 6. 合并

PR 被批准后，维护者会合并您的代码。

## 🐛 问题反馈

### Bug 报告

使用 GitHub Issues 报告 Bug，包含以下信息：

```markdown
**描述 Bug**
清晰简洁地描述 Bug

**复现步骤**
1. 执行 '...'
2. 点击 '...'
3. 看到错误

**期望行为**
描述您期望发生的行为

**实际行为**
描述实际发生了什么

**截图/日志**
如果适用，添加截图或日志

**环境信息**
- OS: [e.g. macOS 13.0]
- Go 版本: [e.g. 1.25]
- Asynq-Hub 版本: [e.g. v1.0.0]
- PostgreSQL 版本: [e.g. 18]
- Redis 版本: [e.g. 7.2]

**额外信息**
其他相关信息
```

### 功能请求

```markdown
**功能描述**
清晰简洁地描述您想要的功能

**使用场景**
描述这个功能的使用场景和价值

**可能的实现**
如果有想法，描述可能的实现方案

**替代方案**
描述您考虑过的替代方案

**额外信息**
其他相关信息
```

## 📚 文档贡献

### 文档类型

1. **API 文档**: 使用 Swagger 注释
2. **代码注释**: Go 和 TypeScript 注释
3. **README**: 项目说明
4. **架构文档**: 系统设计
5. **使用指南**: 教程和示例

### 文档规范

```go
// ✅ 好的 Swagger 注释
// CreateTask 创建新任务
// @Summary 创建任务
// @Description 创建一个新的任务并加入队列
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body dto.CreateTaskRequest true "任务信息"
// @Success 200 {object} dto.TaskResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
    // ...
}
```

### 更新文档

```bash
# 生成 Swagger 文档
make swagger

# 查看文档
open http://localhost:28080/swagger/index.html
```

## 🧪 测试

### 编写测试

```go
// ✅ 好的测试
func TestWorkerRegistration(t *testing.T) {
    // Arrange
    config := WorkerConfig{
        WorkerName: "test-worker",
        BaseURL:    "http://localhost:28080",
    }
    
    // Act
    worker := NewWorker(config)
    err := worker.register()
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, worker)
}

// 测试边界情况
func TestWorkerRegistration_InvalidConfig(t *testing.T) {
    config := WorkerConfig{
        WorkerName: "", // 无效配置
    }
    
    worker := NewWorker(config)
    err := worker.register()
    
    assert.Error(t, err)
}
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test ./internal/worker/...

# 运行测试并查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🎯 开发技巧

### 调试

```bash
# 启用详细日志
LOG_LEVEL=debug make run

# 使用 Delve 调试
dlv debug cmd/server/main.go
```

### 性能分析

```bash
# CPU 性能分析
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# 内存性能分析
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### 常用命令

```bash
# 查看可用命令
make help

# 构建
make build

# 清理
make clean

# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test

# 构建 Docker 镜像
make docker-build

# 生成 Swagger 文档
make swagger
```

## 📞 获取帮助

如果您有任何问题：

1. 查看 [文档](docs/)
2. 搜索 [现有 Issues](https://github.com/azhengyongqin/asynq-hub/issues)
3. 加入 [GitHub Discussions](https://github.com/azhengyongqin/asynq-hub/discussions)
4. 创建新的 Issue

## 🙏 致谢

感谢所有为 Asynq-Hub 做出贡献的开发者！

您的贡献让 Asynq-Hub 变得更好！

---

<div align="center">
Made with ❤️ by Asynq-Hub Team
</div>
