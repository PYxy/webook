package repository

import (
	"context"

	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
)

var (
	ErrFrequentlyForSend      = cache.ErrFrequentlyForSend
	ErrUnknownForCode         = cache.ErrUnknownForCode
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
	ErrAttack                 = cache.ErrAttack
	ErrKnow                   = cache.ErrUnknown
)

type CodeRepository interface {
	Set(ctx context.Context, biz, phone, code string, cnt int) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

type CacheCodeRepository struct {
	cache cache.SmsCache
}

func NewCodeRepository(cache cache.SmsCache) CodeRepository {
	return &CacheCodeRepository{
		cache: cache,
	}
}

func (c *CacheCodeRepository) Set(ctx context.Context, biz, phone, code string, cnt int) error {
	//键：验证码
	//cnt  最多尝试次数
	return c.cache.Set(ctx, biz, phone, code, cnt)
}

func (c *CacheCodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	return c.cache.Verify(ctx, biz, phone, code)

}
