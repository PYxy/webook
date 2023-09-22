package homework

import (
	"context"
	"fmt"
	"math/rand"
	at "sync/atomic"
	"time"

	"go.uber.org/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

var (
	//默认成功处理一个请求就加一分
	incr uint32 = 3
)

func init() {
	rand.Seed(time.Now().Unix())
}

type MonitorService struct {
	//标签  用于监控的识别 如Prometheus的标识字段
	name string
	//监控的服务对象
	svs sms.Service
	//健康状态
	status *atomic.Bool
	//流量承接情况
	//1.如果用随机数 不能说一直拒绝请求
	percentage *atomic.Uint32
	//最大拒绝请求
	MaxMissReq uint32
	//当前拒绝请求
	CurrMissReq uint32

	//可以增加一个检测状态的接口
	//TODO 判断出问题之后
	//1.修改健康状态
	//2.开启异步任务 定时监控相关指标
	//3.按需 修改健康状态 并修改percentage 的承接流量情况
	interval time.Duration
}

func NewMonitorService(sms sms.Service, MaxMissReq, CurrMissReq uint32) sms.Service {
	return &MonitorService{
		name:       "短信测试1",
		svs:        sms,
		status:     atomic.NewBool(true),
		percentage: atomic.NewUint32(100),
		MaxMissReq: 5,
	}
}

// Send
// 1.
// 使用连续多少个异常来判断 缺点 一好一坏 就体验很差了 而且并发太搞的 话 第一个准备重置为0 后面连续5个异常 就会被第一个人给覆盖了
// 2.
func (m *MonitorService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) (err error) {
	//TODO implement me
	//panic("implement me")
	var do bool
	do, err = m.Healthy()
	if do {
		//正常设备接收流量
		//TODO 1全功率的健康设备
		if m.percentage.Load() >= 100 {
			start := time.Now()
			err = m.svs.Send(ctx, biz, phoneNumbers, args)
			costTime := time.Now().Sub(start).Milliseconds()
			m.Collect(MonitorCollect{
				err:        err,
				handleTime: costTime,
			})
			return err
		}
		//TODO 2 异常变正常  需要慢慢预热
		//随机数
		if uint32(rand.Int31n(101)) <= m.percentage.Load() {
			//这里不刷新m.CurrMissReq 是为了 尽快 把设备变成完全健康
			err = m.doSend(ctx, biz, phoneNumbers, args)
			if err != nil {
				//正常处理需要加分 加 3 分
				m.percentage.Add(incr)
			}
			return err
		}
		//前面已经拒绝很多了,不能再拒绝了
		if at.LoadUint32(&m.CurrMissReq) >= m.MaxMissReq {
			//这里应该马上刷新 m.CurrMissReq
			//TODO 1.这里貌似不会有并发问题 (实际上跳过的请求会 >=m.MaxMissReq ?)
			at.StoreUint32(&m.CurrMissReq, 0)
			//不能在拒绝了
			err = m.doSend(ctx, biz, phoneNumbers, args)
			if err != nil {
				//正常处理需要加分 加 3 分
				m.percentage.Add(3)
			}
			return err

		}
		//太虚弱了  暂时不接收流量
		at.AddUint32(&m.CurrMissReq, 1)
		return ErrWeakToDo
	}

	//异常设备的小量放流量 去检查是否变回正常
	// 有一些不需要放小流量去测试 直接 使用监控他cpu\内存\io 之类的指标就行
	
	//少量放流量 的操作是尽可能都把流量放到通过各实例 保证 最快速度把 一个坏的实例 变成好的实例

	//但是 测试异常后 下次就要跳过这个异常实例了  除非只有一个异常的

	return err
}

func (m *MonitorService) doSend(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) (err error) {
	start := time.Now()
	err = m.svs.Send(ctx, biz, phoneNumbers, args)
	costTime := time.Now().Sub(start).Milliseconds()
	m.Collect(MonitorCollect{
		err:        err,
		handleTime: costTime,
	})
	return err
}

// Healthy 判断服务的状态
func (m *MonitorService) Healthy() (bool, error) {
	//可以放在Prometheus 然后通过 请求的方式 获取到指标数据 然后判断
	return m.status.Load(), nil
}

// collect 收集服务的处理结果
func (m *MonitorService) Collect(mc MonitorCollect) {
	//判断MonitorCollect 的结构体然后晒到 收集器中
}

// Statistics 统计数据 判断实例是不是正常的
// 1.当前是本地的滑动窗口去处理
// 2.可以放在Prometheus 然后通过 请求的方式 获取到指标数据 然后判断
func (m *MonitorService) Statistics() {

	//根据实际情况修改服务状态
	//m.status.Store(false)

}

func (m *MonitorService) GetName() string {
	return m.name
}

// Async  每个服务单独一个 还是这个层级 上移?
func (m *MonitorService) Async() {

	go func() {
		ticker := time.NewTicker(m.interval)
		for {
			//这样写就不能控制了
			//<-ticker.C
			select {
			case <-ticker.C:
				do, err := m.Healthy()
				if err != nil {
					fmt.Println(fmt.Sprintf("%s 定时检查失败:%v", m.GetName(), err))
					continue
				}
				if do {
					m.Statistics()
				}

			}
		}
	}()
}
