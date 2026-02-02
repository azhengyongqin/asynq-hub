package workers

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// 默认优先级权重
const (
	PriorityCritical = "critical"
	PriorityDefault  = "default"
	PriorityLow      = "low"
)

// DefaultPriorities 默认优先级权重配置
var DefaultPriorities = map[string]int{
	PriorityCritical: 50,
	PriorityDefault:  30,
	PriorityLow:      10,
}

// QueueGroupConfig 队列组配置
type QueueGroupConfig struct {
	Name        string         `json:"name"`        // 队列组名称
	Concurrency int32          `json:"concurrency"` // 并发数
	Priorities  map[string]int `json:"priorities"`  // 优先级权重
}

// Config 是 worker 配置的完整信息
type Config struct {
	WorkerName        string             `json:"worker_name"`
	BaseURL           string             `json:"base_url,omitempty"`
	RedisAddr         string             `json:"redis_addr,omitempty"`
	QueueGroups       []QueueGroupConfig `json:"queue_groups"` // 队列组配置
	DefaultRetryCount int32              `json:"default_retry_count"`
	DefaultTimeout    int32              `json:"default_timeout"` // seconds
	DefaultDelay      int32              `json:"default_delay"`   // seconds
	IsEnabled         bool               `json:"is_enabled"`
	LastHeartbeatAt   *time.Time         `json:"last_heartbeat_at,omitempty"`
}

// GetQueueGroup 获取指定队列组配置
func (c *Config) GetQueueGroup(queueGroupName string) *QueueGroupConfig {
	for i := range c.QueueGroups {
		if c.QueueGroups[i].Name == queueGroupName {
			return &c.QueueGroups[i]
		}
	}
	return nil
}

// HasQueueGroup 检查是否有指定的队列组
func (c *Config) HasQueueGroup(queueGroupName string) bool {
	return c.GetQueueGroup(queueGroupName) != nil
}

// HasQueueWithPriority 检查是否有指定的队列组和优先级
func (c *Config) HasQueueWithPriority(queueGroupName, priority string) bool {
	qg := c.GetQueueGroup(queueGroupName)
	if qg == nil {
		return false
	}
	_, ok := qg.Priorities[priority]
	return ok
}

// FullQueueName 生成完整队列名：workerName:queueGroupName:priority
func (c *Config) FullQueueName(queueGroupName, priority string) string {
	return fmt.Sprintf("%s:%s:%s", c.WorkerName, queueGroupName, priority)
}

// QueueGroupNames 返回所有队列组名称
func (c *Config) QueueGroupNames() []string {
	names := make([]string, len(c.QueueGroups))
	for i, qg := range c.QueueGroups {
		names[i] = qg.Name
	}
	return names
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

	// 验证 QueueGroups
	if len(c.QueueGroups) == 0 {
		return Config{}, errors.New("queue_groups 不能为空")
	}

	// 验证并设置每个队列组的默认值
	for i := range c.QueueGroups {
		qg := &c.QueueGroups[i]
		if qg.Name == "" {
			return Config{}, errors.New("queue_group name 不能为空")
		}
		// 设置默认并发数
		if qg.Concurrency <= 0 {
			qg.Concurrency = 10
		}
		// 设置默认优先级权重
		if len(qg.Priorities) == 0 {
			qg.Priorities = make(map[string]int)
			for k, v := range DefaultPriorities {
				qg.Priorities[k] = v
			}
		}
	}

	// 设置默认值
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

// HasQueue 检查 worker 是否有指定的队列组
func (s *Store) HasQueue(workerName, queueGroupName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, ok := s.items[workerName]
	if !ok {
		return false
	}

	return worker.HasQueueGroup(queueGroupName)
}

// HasQueueWithPriority 检查 worker 是否有指定的队列组和优先级
func (s *Store) HasQueueWithPriority(workerName, queueGroupName, priority string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, ok := s.items[workerName]
	if !ok {
		return false
	}

	return worker.HasQueueWithPriority(queueGroupName, priority)
}

// GetQueueGroup 获取指定 worker 的指定队列组配置
func (s *Store) GetQueueGroup(workerName, queueGroupName string) (*QueueGroupConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, ok := s.items[workerName]
	if !ok {
		return nil, false
	}

	qg := worker.GetQueueGroup(queueGroupName)
	return qg, qg != nil
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
