# Prisma 数据库管理

本项目使用 Prisma 作为数据库 ORM 和迁移工具。

## 快速开始

### 1. 安装依赖

```bash
pnpm install
```

### 2. 配置数据库连接

在项目根目录的 `.env` 文件中设置数据库连接（或使用 `POSTGRES_DSN` 环境变量）：

```env
DATABASE_URL="postgresql://postgres:postgres@localhost:15432/taskpm?sslmode=disable"
```

或者使用 `POSTGRES_DSN`：

```env
POSTGRES_DSN="postgresql://postgres:postgres@localhost:15432/taskpm?sslmode=disable"
```

### 3. 启动数据库

```bash
docker-compose up -d postgres
```

### 4. 运行迁移

```bash
# 开发环境：创建并应用迁移
pnpm prisma migrate dev

# 生产环境：只应用迁移
pnpm prisma migrate deploy
```

### 5. 生成 Prisma Client

```bash
pnpm prisma generate
```

## 常用命令

### 查看数据库

```bash
# 打开 Prisma Studio（可视化数据库管理工具）
pnpm prisma studio
```

### 创建新迁移

```bash
# 修改 prisma/schema.prisma 后运行
pnpm prisma migrate dev --name <migration_name>
```

### 重置数据库

```bash
# ⚠️ 警告：这会删除所有数据
pnpm prisma migrate reset
```

### 从现有数据库生成 Schema

```bash
pnpm prisma db pull
```

### 格式化 Schema

```bash
pnpm prisma format
```

## 数据模型

### TaskTypeConfig - 任务类型配置

存储每个任务类型的配置信息（优先级、队列、重试等）。

```prisma
model TaskTypeConfig {
  id                 BigInt   @id @default(autoincrement())
  taskName           String   @unique
  priority           Int      @default(0)
  defaultQueue       String   @default("default")
  maxRetry           Int      @default(3)
  timeoutSeconds     Int      @default(30)
  concurrencyWeight  Int      @default(1)
  isEnabled          Boolean  @default(true)
  description        String?
  createdAt          DateTime @default(now())
  updatedAt          DateTime @updatedAt
}
```

### Task - 任务记录

存储每个任务的状态和执行情况，用于分析统计。

```prisma
model Task {
  id              BigInt   @id @default(autoincrement())
  taskId          String   @unique
  taskName        String
  queue           String
  priority        Int
  payload         Json
  status          String   // pending/running/success/fail/dead
  lastAttempt     Int      @default(0)
  lastError       String?
  lastWorkerName  String?
  traceId         String?
  createdAt       DateTime @default(now())
  updatedAt       DateTime @updatedAt
}
```

### TaskAttempt - 任务执行尝试

记录每次任务执行的详细信息（包括重试）。

```prisma
model TaskAttempt {
  id           BigInt    @id @default(autoincrement())
  taskId       String
  asynqTaskId  String?
  attempt      Int
  status       String    // running/success/fail/dead
  startedAt    DateTime  @default(now())
  finishedAt   DateTime?
  durationMs   Int?
  error        String?
  workerName   String?
  traceId      String?
  spanId       String?
}
```

## 在代码中使用 Prisma

### TypeScript/JavaScript

```typescript
import { PrismaClient } from '../generated/prisma'

const prisma = new PrismaClient()

// 查询任务类型配置
const taskTypes = await prisma.taskTypeConfig.findMany({
  where: { isEnabled: true }
})

// 创建任务记录
const task = await prisma.task.create({
  data: {
    taskId: 'unique-task-id',
    taskName: 'example-task',
    queue: 'default',
    priority: 500,
    payload: { url: 'https://example.com' },
    status: 'pending'
  }
})

// 查询任务及其尝试记录
const taskWithAttempts = await prisma.task.findUnique({
  where: { taskId: 'unique-task-id' },
  include: { attempts: true }
})

// 更新任务状态
await prisma.task.update({
  where: { taskId: 'unique-task-id' },
  data: { 
    status: 'success',
    lastAttempt: 1,
    updatedAt: new Date()
  }
})
```

### 在 Go 后端中使用

虽然 Prisma Client 主要用于 TypeScript/JavaScript，但你可以：

1. **继续使用现有的 Go 代码**：保持 `backend/internal/repository/postgresrepo` 的实现
2. **使用 Prisma 管理迁移**：只用 Prisma 做 schema 定义和迁移管理
3. **混合使用**：前端/管理工具用 Prisma Client，后端用 pgx

## 迁移策略

### 从旧的 SQL 迁移到 Prisma

1. **保留旧的迁移文件**：`backend/migrations/` 中的 SQL 文件作为历史记录
2. **使用 Prisma 管理新迁移**：从现在开始，所有新的 schema 变更都通过 Prisma
3. **数据兼容性**：Prisma 生成的表结构与旧的 SQL 完全兼容

### 迁移文件位置

- **Prisma 迁移**：`prisma/migrations/`
- **旧的 SQL 迁移**：`backend/migrations/`（保留作为历史记录）

## 最佳实践

1. **总是先修改 schema.prisma**，然后运行 `migrate dev`
2. **提交迁移文件到 Git**：`prisma/migrations/` 目录应该被版本控制
3. **生产环境使用 `migrate deploy`**：不要在生产环境使用 `migrate dev`
4. **定期备份数据库**：在执行迁移前
5. **使用事务**：对于复杂的数据操作，使用 Prisma 的事务功能

## 故障排查

### 连接失败

```bash
# 检查数据库是否运行
docker-compose ps postgres

# 检查数据库是否就绪
docker-compose exec postgres pg_isready -U postgres

# 查看数据库日志
docker-compose logs postgres
```

### 迁移失败

```bash
# 查看迁移状态
pnpm prisma migrate status

# 标记迁移为已应用（如果手动执行了 SQL）
pnpm prisma migrate resolve --applied <migration_name>

# 回滚到特定迁移（需要手动操作）
# Prisma 不支持自动回滚，需要手动编写回滚 SQL
```

### Schema 不同步

```bash
# 查看差异
pnpm prisma migrate diff \
  --from-schema-datamodel prisma/schema.prisma \
  --to-schema-datasource prisma/schema.prisma

# 重新生成 Client
pnpm prisma generate
```

## 相关资源

- [Prisma 官方文档](https://www.prisma.io/docs)
- [Prisma Schema 参考](https://www.prisma.io/docs/reference/api-reference/prisma-schema-reference)
- [Prisma Client API](https://www.prisma.io/docs/reference/api-reference/prisma-client-reference)
- [Prisma Migrate](https://www.prisma.io/docs/concepts/components/prisma-migrate)
