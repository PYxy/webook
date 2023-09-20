package retryable

import (
	"context"
	"errors"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

// Service 存在并发问题
type Service struct {
	svc sms.Service
	// 重试
	retryMax int
}

func NewService(svc sms.Service, retryMax int) sms.Service {
	return &Service{
		svc:      svc,
		retryMax: retryMax,
	}
}

func (s *Service) Send(ctx context.Context, tpl string, numbers []string, args []sms.ArgVal) error {
	err := s.svc.Send(ctx, tpl, numbers, args)
	cnt := 1
	for err != nil && cnt < s.retryMax {
		err = s.svc.Send(ctx, tpl, numbers, args)
		if err == nil {
			return nil
		}
		cnt++
	}
	return errors.New("重试都失败了")
}
