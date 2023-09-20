package homework

import "gitee.com/geekbang/basic-go/webook/internal/service/sms"

// OPERATOR 根据当前服务李彪是不是存在异常设备 有就进行分流检查
type NewService interface {
	sms.Service
	Healthy() bool
}
