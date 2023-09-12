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

func (s *Service) Send(ctx context.Context, tpl string, phoneNumbers []string, args []sms.ArgVal) error {
	for _, val := range args {
		fmt.Printf("业务类型： %s, key：%s, val ：%s \n", tpl, val.Name, val.Val)
	}
	return nil
}
