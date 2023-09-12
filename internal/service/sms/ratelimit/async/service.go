package async

import (
	"context"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type SyncSMSService struct {
	svc sms.Service
	//repo   数据库连接对象用于保存失败的短信请求
}

func (s *SyncSMSService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	err := s.svc.Send(ctx, biz, phoneNumbers, args)
	if err != nil {
		//判断错误类型 是否需要重试
		//按需保存
		//使用repo 保存
	}

	return nil
}

// Async 异步捞数据
func (s *SyncSMSService) Async() {
	go func() {

		//1.使用repo 对象查询数据库有没有任务
		//2.拉出来然后发送
		//s.svc.Send()
		//3.延时
	}()

}
