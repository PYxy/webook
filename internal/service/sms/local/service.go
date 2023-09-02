package local

//本地短信服务 假发  只是打印

import (
	"context"
	"fmt"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type Service struct {
}

func NewLocalSmsService() sms.Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, phoneNumbers []string, args []sms.ArgVal) error {
	for _, val := range args {
		fmt.Printf("key：%s, val ：%s", val.Name, val.Val)
	}
	return nil
}
