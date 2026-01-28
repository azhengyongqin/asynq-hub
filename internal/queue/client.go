package asynqx

import "github.com/hibiken/asynq"

type Client struct {
	*asynq.Client
}

func NewClient(redisAddr string) *Client {
	opt, err := NewRedisConnOpt(redisAddr)
	if err != nil {
		// 这里是内部封装，保持接口简单：配置非法直接 panic（由上层 main 负责校验更友好的错误）
		panic(err)
	}
	return &Client{Client: asynq.NewClient(opt)}
}
