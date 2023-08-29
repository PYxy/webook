package sms

import "context"

//短信发送服务 是验证服务里面的一部分

type Service interface {
	Send(ctx context.Context, phoneNumbers []string, args []ArgVal) error
}

type ArgVal struct {
	Val  string
	Name string
}
