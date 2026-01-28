-- CreateTable
CREATE TABLE "worker" (
    "id" BIGSERIAL NOT NULL,
    "worker_name" TEXT NOT NULL,
    "base_url" TEXT,
    "redis_addr" TEXT,
    "concurrency" INTEGER NOT NULL DEFAULT 10,
    "queues" JSONB NOT NULL,
    "default_retry_count" INTEGER NOT NULL DEFAULT 3,
    "default_timeout" INTEGER NOT NULL DEFAULT 30,
    "default_delay" INTEGER NOT NULL DEFAULT 0,
    "is_enabled" BOOLEAN NOT NULL DEFAULT true,
    "last_heartbeat_at" TIMESTAMPTZ(6),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "worker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "task" (
    "id" BIGSERIAL NOT NULL,
    "task_id" TEXT NOT NULL,
    "worker_name" TEXT NOT NULL,
    "queue" TEXT NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "payload" JSONB NOT NULL,
    "status" TEXT NOT NULL,
    "last_attempt" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "last_worker_name" TEXT,
    "trace_id" TEXT,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "task_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "task_attempt" (
    "id" BIGSERIAL NOT NULL,
    "task_id" TEXT NOT NULL,
    "asynq_task_id" TEXT,
    "attempt" INTEGER NOT NULL,
    "status" TEXT NOT NULL,
    "started_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "finished_at" TIMESTAMPTZ(6),
    "duration_ms" INTEGER,
    "error" TEXT,
    "worker_name" TEXT,
    "trace_id" TEXT,
    "span_id" TEXT,

    CONSTRAINT "task_attempt_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "worker_worker_name_key" ON "worker"("worker_name");

-- CreateIndex
CREATE INDEX "idx_worker_enabled" ON "worker"("is_enabled");

-- CreateIndex
CREATE INDEX "idx_worker_heartbeat" ON "worker"("last_heartbeat_at");

-- CreateIndex
CREATE UNIQUE INDEX "task_task_id_key" ON "task"("task_id");

-- CreateIndex
CREATE INDEX "idx_task_worker_created_at" ON "task"("worker_name", "created_at" DESC);

-- CreateIndex
CREATE INDEX "idx_task_status_updated_at" ON "task"("status", "updated_at" DESC);

-- CreateIndex
CREATE INDEX "idx_task_queue_updated_at" ON "task"("queue", "updated_at" DESC);

-- CreateIndex
CREATE INDEX "idx_attempt_task_key_started_at" ON "task_attempt"("task_id", "started_at" DESC);

-- CreateIndex
CREATE INDEX "idx_attempt_status_started_at" ON "task_attempt"("status", "started_at" DESC);

-- AddForeignKey
ALTER TABLE "task" ADD CONSTRAINT "task_worker_name_fkey" FOREIGN KEY ("worker_name") REFERENCES "worker"("worker_name") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "task_attempt" ADD CONSTRAINT "task_attempt_task_id_fkey" FOREIGN KEY ("task_id") REFERENCES "task"("task_id") ON DELETE RESTRICT ON UPDATE CASCADE;
