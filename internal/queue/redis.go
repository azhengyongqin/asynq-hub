package asynqx

import "github.com/hibiken/asynq"

// NewRedisConnOpt 仅接受 URI（例如 redis://localhost:6379/6）。
// 统一用 asynq.ParseRedisURI，避免手工拆分 addr/db。
func NewRedisConnOpt(redisURI string) (asynq.RedisConnOpt, error) {
	return asynq.ParseRedisURI(redisURI)
}
