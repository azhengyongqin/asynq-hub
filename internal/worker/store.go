package workers

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config 是 worker 配置的完整信息
type Config struct {
	WorkerName        string         `json:"worker_name"`
	BaseURL           string         `json:"base_url,omitempty"`
	RedisAddr         string         `json:"redis_addr,omitempty"`
	Concurrency       int32          `json:"concurrency"`
	Queues            map[string]int `json:"queues"` // queue_name -> weight
	DefaultRetryCount int32          `json:"default_retry_count"`
	DefaultTimeout    int32          `json:"default_timeout"` // seconds
	DefaultDelay      int32          `json:"default_delay"`   // seconds
	IsEnabled         bool           `json:"is_enabled"`
	LastHeartbeatAt   *time.Time     `json:"last_heartbeat_at,omitempty"`
}

// QueueConfig 队列的配置信息
type QueueConfig struct {
	QueueName string `json:"queue_name"`
	Weight    int    `json:"weight"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string]Config // key: worker_name
}

func NewStore() *Store {
	return &Store{
		items: map[string]Config{},
	}
}

// List 返回所有 worker 配置
func (s *Store) List() []Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Config, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, v)
	}

	// 按 worker_name 排序
	sort.Slice(out, func(i, j int) bool {
		return out[i].WorkerName < out[j].WorkerName
	})

	return out
}

// Get 获取指定 worker 的配置
func (s *Store) Get(workerName string) (Config, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[workerName]
	return v, ok
}

// Upsert 创建或更新 worker 配置
func (s *Store) Upsert(c Config) (Config, error) {
	c.WorkerName = strings.TrimSpace(c.WorkerName)
	if c.WorkerName == "" {
		return Config{}, errors.New("worker_name 不能为空")
	}

	// 验证 Queues
	if len(c.Queues) == 0 {
		return Config{}, errors.New("queues 不能为空")
	}

	// 设置默认值
	if c.Concurrency <= 0 {
		c.Concurrency = 10
	}
	if c.DefaultRetryCount <= 0 {
		c.DefaultRetryCount = 3
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 30
	}
	if c.DefaultDelay < 0 {
		c.DefaultDelay = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[c.WorkerName] = c
	return c, nil
}

// GetQueueConfig 获取指定 worker 的指定队列配置
func (s *Store) GetQueueConfig(workerName, queueName string) (QueueConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, ok := s.items[workerName]
	if !ok {
		return QueueConfig{}, false
	}

	weight, ok := worker.Queues[queueName]
	if !ok {
		return QueueConfig{}, false
	}

	return QueueConfig{
		QueueName: queueName,
		Weight:    weight,
	}, true
}

// HasQueue 检查 worker 是否有指定的队列
func (s *Store) HasQueue(workerName, queueName string) bool {
	_, ok := s.GetQueueConfig(workerName, queueName)
	return ok
}

// UpdateHeartbeat 更新 worker 的心跳时间
func (s *Store) UpdateHeartbeat(workerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	worker, ok := s.items[workerName]
	if !ok {
		return errors.New("worker 不存在")
	}

	now := time.Now()
	worker.LastHeartbeatAt = &now
	s.items[workerName] = worker
	return nil
}

// Delete 删除指定 worker
func (s *Store) Delete(workerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[workerName]; !ok {
		return errors.New("worker 不存在")
	}

	delete(s.items, workerName)
	return nil
}
