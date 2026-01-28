package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Upsert(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName:        "test-worker",
		Concurrency:       10,
		Queues:            map[string]int{"default": 5},
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
	assert.Equal(t, cfg.Concurrency, retrieved.Concurrency)

	// 测试更新
	cfg.Concurrency = 20
	_, err = store.Upsert(cfg)
	assert.NoError(t, err, "更新应该成功")

	// 验证更新
	retrieved, ok = store.Get("test-worker")
	require.True(t, ok)
	assert.Equal(t, int32(20), retrieved.Concurrency, "并发度应该更新为20")
}

func TestStore_Delete(t *testing.T) {
	store := NewStore()

	cfg := Config{
		WorkerName: "test-worker",
		Queues:     map[string]int{"default": 5},
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
	store.Upsert(Config{WorkerName: "worker1", Queues: map[string]int{"default": 5}})
	store.Upsert(Config{WorkerName: "worker2", Queues: map[string]int{"default": 5}})
	store.Upsert(Config{WorkerName: "worker3", Queues: map[string]int{"default": 5}})

	list = store.List()
	assert.Len(t, list, 3, "应该有3个配置")
}

func TestStore_Validate(t *testing.T) {
	store := NewStore()

	t.Run("empty worker name", func(t *testing.T) {
		_, err := store.Upsert(Config{Queues: map[string]int{"default": 5}})
		assert.Error(t, err)
	})

	t.Run("empty queues", func(t *testing.T) {
		_, err := store.Upsert(Config{WorkerName: "test"})
		assert.Error(t, err)
	})

	t.Run("valid config", func(t *testing.T) {
		_, err := store.Upsert(Config{
			WorkerName: "test",
			Queues:     map[string]int{"default": 5},
		})
		assert.NoError(t, err)
	})
}
