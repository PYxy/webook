package ioc

import (
	"context"
	"time"

	v9 "github.com/redis/go-redis/v9"

	"gitee.com/geekbang/basic-go/webook/config"
)

func InitRedis() v9.Cmdable {
	cache := v9.NewClient(&v9.Options{
		Addr:     config.Config.Redis.Addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	err := cache.Conn().Ping(ctx).Err()
	if err != nil {
		panic("redis  连接失败")
	}
	return cache
}
