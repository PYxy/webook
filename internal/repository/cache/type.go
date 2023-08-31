package cache

import (
	"context"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
)

type SmsCache interface {
	Set(ctx context.Context, biz, phone, code string, cnt int) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
	GenerateKey(biz, phone string) string
}

type UserCache interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, u domain.User) error
}
