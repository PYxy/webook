package cache

import "context"

type Cache interface {
	Set(ctx context.Context, biz, phone, code string, cnt int) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
	GenerateKey(biz, phone string) string
}
