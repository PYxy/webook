package homework

import (
	"context"
	"errors"

	"go.uber.org/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type MonitorService struct {
	name string
	//监控的服务对象
	svs sms.Service
	//健康状态
	status *atomic.Bool
	//流量承接情况
	percentage *atomic.Uint32
	//可以增加一个检测状态的接口
	//TODO 判断出问题之后
	//1.修改健康状态
	//2.开启异步任务 定时监控相关指标
	//3.按需 修改健康状态 并修改percentage 的承接流量情况

}

func NewMonitorService(sms sms.Service) sms.Service {
	return &MonitorService{
		name:       "短信测试1",
		svs:        sms,
		status:     atomic.NewBool(true),
		percentage: atomic.NewUint32(100),
	}
}

func (m *MonitorService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO implement me

	//panic("implement me")
	if m.Healthy() {
		//健康走 流量承载的路 有可能不接的 这个ctx 基本上没有消耗

		return errors.New("异常变健康,只接受少量流量")
	}
	//异常设备的小量放流量 去检查是否变回正常
	//少量放流量 的操作是尽可能都把流量放到通过各实例 保证 最快速度把 一个坏的实例 放到好的实例中
	//但是 测试异常后 下次就要跳过这个异常实例了  除非只有一个异常的
}

func (m *MonitorService) Healthy() bool {
	return m.status.Load()
}
