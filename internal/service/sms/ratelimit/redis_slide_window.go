package ratelimit

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed slide_window.lua
var luaSlideWindow string

type RedisSlidingWindowLimiter struct {
	cmd redis.Cmdable
	//检查间隔
	interval time.Duration
	//阈值
	rate int
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, duration time.Duration, rate int) *RedisSlidingWindowLimiter {

	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		interval: duration,
		rate:     rate,
	}
}

func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {

	return r.cmd.Eval(ctx, luaSlideWindow, []string{key}, r.interval.Microseconds(), r.rate, time.Now().UnixMilli()).Bool()
}
