-- 迁移：将 queues 字段转换为 queue_groups 结构
-- 这是一个破坏性变更，需要重新配置所有 Worker

-- 1. 删除旧的 concurrency 列（因为每个队列组有独立的并发数）
ALTER TABLE "worker" DROP COLUMN IF EXISTS "concurrency";

-- 2. 将 queues 字段重命名为 queue_groups
ALTER TABLE "worker" RENAME COLUMN "queues" TO "queue_groups";

-- 3. 添加注释说明新的数据结构
COMMENT ON COLUMN "worker"."queue_groups" IS 'JSON数组，每个元素包含: name(队列组名称), concurrency(并发数), priorities(优先级权重map)';

-- 示例数据结构:
-- [
--   {
--     "name": "web_crawl",
--     "concurrency": 10,
--     "priorities": {
--       "critical": 50,
--       "default": 30,
--       "low": 10
--     }
--   }
-- ]

-- 注意: 由于数据结构变更，需要在应用迁移后清理旧数据或手动转换
-- 建议: 先备份数据，然后清空 worker 表，让 Worker 重新注册
