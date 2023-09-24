package dao

import (
	"context"
	"gorm.io/gorm"
)

type SmsMsg struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	//异步任务有没有发送成功
	status bool
	//激活状态
	Active bool

	// 创建时间，毫秒数
	Ctime int64
	// 更新时间，毫秒数
	Utime int64

	Biz string
	//短信发送参数
	PhoneNumbers string //前缀索引
	Args         string
}
type SMSDAO struct {
	db *gorm.DB
}

func NewSmsDao(db *gorm.DB) SMSDaoInterface {
	return &SMSDAO{db: db}
}

func (s *SMSDAO) Select(ctx context.Context, status bool, active bool) ([]SmsMsg, error) {
	//TODO implement me
	//先把超过短信重发限制的短信剔除
	panic("implement me")
}

func (s *SMSDAO) Insert(ctx context.Context, u SmsMsg) error {
	//TODO implement me
	panic("implement me")
}

func (s *SMSDAO) Update(ctx context.Context, id int, status, active bool) error {
	//TODO implement me
	panic("implement me")
}
