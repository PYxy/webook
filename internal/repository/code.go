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
)

type CodeRepository struct {
	cache cache.Cache
}

func NewCodeRepository(cache cache.Cache) *CodeRepository {
	return &CodeRepository{
		cache: cache,
	}
}

func (c *CodeRepository) Set(ctx context.Context, biz, phone, code string, cnt int) error {
	//键：验证码
	//cnt  最多尝试次数
	return c.cache.Set(ctx, biz, phone, code, cnt)
}

func (c *CodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	return c.cache.Verify(ctx, biz, phone, code)

}
