package ratelimit

import "context"

type Limiter interface {
	//Limit 有没有触发限流
	//key 是限流对象
	//bool true 表示要限流
	//err 限流器本身有没有错误
	Limit(ctx context.Context, key string) (bool, error)
}
