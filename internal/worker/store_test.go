package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Upsert(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName: "test-worker",
		QueueGroups: []QueueGroupConfig{
			{
				Name:        "default",
				Concurrency: 10,
				Priorities:  DefaultPriorities,
			},
		},
		DefaultRetryCount: 3,
		DefaultTimeout:    30,
	}

	// 测试插入
	_, err := store.Upsert(cfg)
	assert.NoError(t, err, "插入应该成功")

	// 验证存储
	retrieved, ok := store.Get("test-worker")
	require.True(t, ok, "应该能获取刚插入的配置")
	assert.Equal(t, cfg.WorkerName, retrieved.WorkerName)
	assert.Len(t, retrieved.QueueGroups, 1)
	assert.Equal(t, int32(10), retrieved.QueueGroups[0].Concurrency)

	// 测试更新
	cfg.QueueGroups[0].Concurrency = 20
	_, err = store.Upsert(cfg)
	assert.NoError(t, err, "更新应该成功")

	// 验证更新
	retrieved, ok = store.Get("test-worker")
	require.True(t, ok)
	assert.Equal(t, int32(20), retrieved.QueueGroups[0].Concurrency, "并发度应该更新为20")
}

func TestStore_Delete(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName: "test-worker",
		QueueGroups: []QueueGroupConfig{
			{Name: "default", Concurrency: 5},
		},
	}
	store.Upsert(cfg)

	// 测试删除
	err := store.Delete("test-worker")
	assert.NoError(t, err, "应该成功删除")

	// 验证删除
	_, ok := store.Get("test-worker")
	assert.False(t, ok, "删除后不应该能获取")

	// 测试删除不存在的
	err = store.Delete("non-existent")
	assert.Error(t, err, "删除不存在的应该返回错误")
}

func TestStore_List(t *testing.T) {
	store := NewStore()

	// 空列表
	list := store.List()
	assert.Empty(t, list, "初始列表应为空")

	// 添加多个配置
	store.Upsert(Config{WorkerName: "worker1", QueueGroups: []QueueGroupConfig{{Name: "default"}}})
	store.Upsert(Config{WorkerName: "worker2", QueueGroups: []QueueGroupConfig{{Name: "default"}}})
	store.Upsert(Config{WorkerName: "worker3", QueueGroups: []QueueGroupConfig{{Name: "default"}}})

	list = store.List()
	assert.Len(t, list, 3, "应该有3个配置")
}

func TestStore_Validate(t *testing.T) {
	store := NewStore()

	t.Run("empty worker name", func(t *testing.T) {
		_, err := store.Upsert(Config{QueueGroups: []QueueGroupConfig{{Name: "default"}}})
		assert.Error(t, err)
	})

	t.Run("empty queue groups", func(t *testing.T) {
		_, err := store.Upsert(Config{WorkerName: "test"})
		assert.Error(t, err)
	})

	t.Run("valid config", func(t *testing.T) {
		_, err := store.Upsert(Config{
			WorkerName: "test",
			QueueGroups: []QueueGroupConfig{
				{Name: "default", Concurrency: 5},
			},
		})
		assert.NoError(t, err)
	})
}

func TestStore_HasQueue(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName: "test-worker",
		QueueGroups: []QueueGroupConfig{
			{
				Name:        "web_crawl",
				Concurrency: 10,
				Priorities:  DefaultPriorities,
			},
			{
				Name:        "data_process",
				Concurrency: 8,
				Priorities:  DefaultPriorities,
			},
		},
	}
	store.Upsert(cfg)

	// 测试队列组存在
	assert.True(t, store.HasQueue("test-worker", "web_crawl"))
	assert.True(t, store.HasQueue("test-worker", "data_process"))

	// 测试队列组不存在
	assert.False(t, store.HasQueue("test-worker", "non_existent"))
	assert.False(t, store.HasQueue("non_existent_worker", "web_crawl"))
}

func TestStore_HasQueueWithPriority(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName: "test-worker",
		QueueGroups: []QueueGroupConfig{
			{
				Name:        "web_crawl",
				Concurrency: 10,
				Priorities:  DefaultPriorities,
			},
		},
	}
	store.Upsert(cfg)

	// 测试优先级存在
	assert.True(t, store.HasQueueWithPriority("test-worker", "web_crawl", PriorityCritical))
	assert.True(t, store.HasQueueWithPriority("test-worker", "web_crawl", PriorityDefault))
	assert.True(t, store.HasQueueWithPriority("test-worker", "web_crawl", PriorityLow))

	// 测试优先级不存在
	assert.False(t, store.HasQueueWithPriority("test-worker", "web_crawl", "invalid_priority"))
	assert.False(t, store.HasQueueWithPriority("test-worker", "non_existent", PriorityCritical))
}

func TestConfig_FullQueueName(t *testing.T) {
	cfg := Config{
		WorkerName: "worker-1",
		QueueGroups: []QueueGroupConfig{
			{Name: "web_crawl", Concurrency: 10, Priorities: DefaultPriorities},
		},
	}

	assert.Equal(t, "worker-1:web_crawl:critical", cfg.FullQueueName("web_crawl", PriorityCritical))
	assert.Equal(t, "worker-1:web_crawl:default", cfg.FullQueueName("web_crawl", PriorityDefault))
	assert.Equal(t, "worker-1:web_crawl:low", cfg.FullQueueName("web_crawl", PriorityLow))
}

func TestConfig_QueueGroupNames(t *testing.T) {
	cfg := Config{
		WorkerName: "worker-1",
		QueueGroups: []QueueGroupConfig{
			{Name: "web_crawl"},
			{Name: "data_process"},
			{Name: "image_process"},
		},
	}

	names := cfg.QueueGroupNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "web_crawl")
	assert.Contains(t, names, "data_process")
	assert.Contains(t, names, "image_process")
}

func TestStore_DefaultPriorities(t *testing.T) {
	store := NewStore()

	// 不提供 Priorities，应该使用默认值
	cfg := Config{
		WorkerName: "test-worker",
		QueueGroups: []QueueGroupConfig{
			{Name: "default", Concurrency: 10},
		},
	}
	result, err := store.Upsert(cfg)
	assert.NoError(t, err)

	// 验证默认优先级已设置
	assert.Len(t, result.QueueGroups[0].Priorities, 3)
	assert.Equal(t, 50, result.QueueGroups[0].Priorities[PriorityCritical])
	assert.Equal(t, 30, result.QueueGroups[0].Priorities[PriorityDefault])
	assert.Equal(t, 10, result.QueueGroups[0].Priorities[PriorityLow])
}
