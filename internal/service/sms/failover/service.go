package failover

import (
	"context"
	"errors"
	"sync/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type FailoverSMSService struct {
	svcs   []sms.Service
	idx    uint64
	length uint64
}

func NewFailoverSMSService(svcs []sms.Service) sms.Service {
	return &FailoverSMSService{
		svcs:   svcs,
		idx:    0,
		length: uint64(len(svcs)),
	}
}

func (f *FailoverSMSService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO implement me

	idx := atomic.AddUint64(&f.idx, 1)
	for i := idx; i < f.length+idx; i++ {
		svc := f.svcs[int(i%f.length)]
		err := svc.Send(ctx, biz, phoneNumbers, args)
		switch err {
		case nil:
			return nil
		case context.Canceled, context.DeadlineExceeded:
			return err
		default:
			//输出错误日志
		}
	}
	return errors.New("全部服务商都发送短信异常")
}
