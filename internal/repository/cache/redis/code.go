package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
)

//go:embed lua/set_code.lua
var luaSetCode string

//go:embed lua/get_code.lua
var luaGetCode string

type CodeRedisCache struct {
	client redis.Cmdable
}

func NewRedisSmsCache(client redis.Cmdable) cache.SmsCache {

	return &CodeRedisCache{
		client: client,
	}
}

func (c *CodeRedisCache) Set(ctx context.Context, biz, phone, code string, cnt int) error {
	//键：验证码
	//cnt  最多尝试次数
	result, err := c.client.Eval(ctx, luaSetCode, []string{c.GenerateKey(biz, phone)}, code, 600, 90, cnt).Int()
	fmt.Println(result, err)
	if err != nil {
		return err
	}
	switch result {
	case 0:
		return nil
	case -1:
		// 请求频繁
		return cache.ErrFrequentlyForSend
	case -2:
		// key 的过期时间异常
		return cache.ErrUnknownForCode
	default:

		return errors.New("系统异常")
	}
}

func (c *CodeRedisCache) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	//fmt.Println(c.GenerateKey(biz, phone))
	res, err := c.client.Eval(ctx, luaGetCode, []string{c.GenerateKey(biz, phone)}, code).Int()
	fmt.Println(res, err)
	if err != nil {
		return false, err
	}
	fmt.Println("...........")
	switch res {
	case 0:
		return true, nil
	case -1:
		// 正常来说，如果频繁出现这个错误，你就要告警，因为有人搞你
		return false, cache.ErrCodeVerifyTooManyTimes
	case -2:
		//用户在合理范围内 密码输入错误
		return false, nil
	case -3:
		//获取验证码之后  修改电话号码(搞事操作)
		return false, cache.ErrAttack
	}
	//未知错误
	fmt.Println("...........")
	return false, cache.ErrUnknown
}

func (c *CodeRedisCache) GenerateKey(biz, phone string) string {

	return fmt.Sprintf("code:%s:%s", biz, phone)
}
