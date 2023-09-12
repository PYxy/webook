package sms

import "context"

//短信发送服务 是验证服务里面的一部分

type Service interface {
	//Send
	//biz  可能是业务类型
	//     也可能是加密后的字符串
	Send(ctx context.Context, biz string, phoneNumbers []string, args []ArgVal) error
	//Send(ctx context.Context,tpl string, phoneNumbers []string, args []ArgVal) error
	//
}

type ArgVal struct {
	Val  string
	Name string
}
