# TasksPage 表头更新

## 更新内容

前端任务监控页面的表头已更新为更清晰的结构。

## 表头变更

### 之前的表头
```
| Task Details (45%) | Status (15%) | Metrics (25%) | Updated (15%) |
```

### 更新后的表头
```
| TaskID (30%) | WorkerName (15%) | Status (15%) | Metrics (25%) | UpdatedAt (15%) |
```

## 详细说明

### 1. **TaskID 列** (30%)
- **显示内容**：
  - 主要：任务唯一标识（`task_id`）
  - 次要：任务类型名称（`task_name`）和队列（`queue`）
- **样式**：
  - 使用等宽字体（`font-mono`）
  - 鼠标悬停时高亮为主色调
  - 超长文本自动截断并显示 tooltip

### 2. **WorkerName 列** (15%) - 新增
- **显示内容**：
  - 最后执行该任务的 Worker 名称（`worker_name`）
  - 如果没有 Worker 信息，显示 "—"
- **样式**：
  - 灰色文字，简洁展示
  - 超长文本自动截断并显示 tooltip

### 3. **Status 列** (15%)
- **显示内容**：
  - 任务状态徽章（`status`）
  - 带有状态图标和颜色
- **状态类型**：
  - `pending` - 灰色
  - `running` - 蓝色
  - `success` - 绿色
  - `fail` / `dead` - 红色

### 4. **Metrics 列** (25%)
- **显示内容**：
  - Priority（优先级）
  - Attempts（尝试次数）
- **样式**：
  - 带图标的紧凑展示
  - 使用 Hash 和 Layers 图标

### 5. **UpdatedAt 列** (15%)
- **显示内容**：
  - 相对时间（如 "2h ago"）
  - 精确时间（如 "14:30"）
- **样式**：
  - 右对齐
  - 两行展示，主次分明

## 列宽分配

| 列名 | 宽度 | 说明 |
|------|------|------|
| TaskID | 30% | 主要信息，需要足够空间显示完整 ID |
| WorkerName | 15% | 新增列，显示 Worker 信息 |
| Status | 15% | 状态徽章，固定宽度 |
| Metrics | 25% | 指标信息，需要空间显示多个指标 |
| UpdatedAt | 15% | 时间信息，右对齐 |

## 数据结构

确保后端 API 返回的任务对象包含以下字段：

```typescript
type Task = {
  task_id: string          // 任务唯一标识
  task_name: string        // 任务类型名称
  queue: string            // 队列名称
  priority: number         // 优先级
  status: string           // 状态
  last_attempt: number     // 最后尝试次数
  last_error?: string      // 最后错误信息
  worker_name?: string     // Worker 名称（新增）
  created_at: string       // 创建时间
  updated_at: string       // 更新时间
}
```

## 视觉效果

### 表头
```
┌─────────────────────────────┬────────────────┬────────────┬─────────────────────────┬────────────────┐
│ TaskID                      │ WorkerName     │ Status     │ Metrics                 │ UpdatedAt      │
├─────────────────────────────┼────────────────┼────────────┼─────────────────────────┼────────────────┤
│ task-abc123                 │ worker-01      │ ● success  │ Pri: 600  Att: 1        │ 2h ago         │
│ crawler.fetch_url • crawler │                │            │                         │ 14:30          │
├─────────────────────────────┼────────────────┼────────────┼─────────────────────────┼────────────────┤
│ task-def456                 │ worker-02      │ ● running  │ Pri: 500  Att: 2        │ Just now       │
│ crawler.parse_page • default│                │            │                         │ 16:45          │
└─────────────────────────────┴────────────────┴────────────┴─────────────────────────┴────────────────┘
```

## 响应式设计

- 在小屏幕上，表格会自动调整
- 使用横向滚动确保所有列都可见
- 表头固定在顶部（sticky header）

## 交互功能

- ✅ 点击任何行打开任务详情弹窗
- ✅ 鼠标悬停时行背景高亮
- ✅ TaskID 悬停时变为主色调
- ✅ 超长文本显示 tooltip

## 兼容性

- ✅ 与现有的 Prisma Schema 完全兼容
- ✅ 支持 `worker_name` 字段（可选）
- ✅ 向后兼容，如果没有 `worker_name` 显示占位符

## 测试建议

1. **有 Worker 信息的任务**：
   - 确认 WorkerName 列正确显示 Worker 名称

2. **无 Worker 信息的任务**：
   - 确认显示 "—" 占位符

3. **不同状态的任务**：
   - 确认状态徽章颜色正确

4. **超长 TaskID**：
   - 确认文本截断和 tooltip 工作正常

5. **响应式布局**：
   - 在不同屏幕尺寸下测试表格显示
