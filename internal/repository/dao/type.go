package dao

import (
	"context"
)

type UserDaoInterface interface {
	FindByEmail(ctx context.Context, email string) (User, error)
	Insert(ctx context.Context, u User) error
	Update(ctx context.Context, u User) error
	FindById(ctx context.Context, id int64) (u User, err error)
	FindByPhone(ctx context.Context, phone string) (u User, err error)
	FindByWeChat(ctx context.Context, openId string) (u User, err error)
}

type SMSDaoInterface interface {
	Select(ctx context.Context, status bool, active bool) ([]SmsMsg, error)
	Insert(ctx context.Context, u SmsMsg) error
	Update(ctx context.Context, id int, status bool, active bool) error
}
