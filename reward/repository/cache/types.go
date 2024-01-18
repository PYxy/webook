package cache

import (
	"context"
	"gitee.com/geekbang/basic-go/webook/reward/domain"
)

type RewardCache interface {
	GetCachedCodeURL(ctx context.Context, r domain.Reward) (domain.CodeURL, error)
	CachedCodeURL(ctx context.Context, cu domain.CodeURL, r domain.Reward) error
}
