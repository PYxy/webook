package homework

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	at "sync/atomic"
	"time"

	"go.uber.org/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

var (
	// ErrWeakToDo 刚从异常变为健康状态 需要慢慢预热
	ErrWeakToDo        = errors.New("虚弱状态")
	ErrNoEnoughToCheck = errors.New("异常设备,流量样本不足")
)

var (
	//默认成功处理一个请求就加一分
	incr uint32 = 3
	//异常状态下 接受5个请求就进行一次 状态检测
	flowReqToHandle uint32 = 5
)

func init() {
	rand.Seed(time.Now().Unix())
}

type MonitorService struct {
	//标签  用于监控的识别 如Prometheus的标识字段
	name string
	//监控的服务对象
	sms.Service
	//健康状态
	status *atomic.Bool
	//平均响应时间(单位:毫秒)
	AverageResp int64
	//长尾时间限制(单位:毫秒)
	LongResp int64
	//流量承接情况
	//1.如果用随机数 不能说一直拒绝请求
	percentage *atomic.Uint32
	//最大拒绝请求
	MaxMissReq uint32
	//当前拒绝请求
	CurrMissReq uint32

	//异常状态中处理的请求
	OnErrReq           uint32
	HappenErrTimeStamp int64

	//可以增加一个检测状态的接口
	//TODO 判断出问题之后
	//1.修改健康状态
	//2.开启异步任务 定时监控相关指标
	//3.按需 修改健康状态 并修改percentage 的承接流量情况
	interval time.Duration

	localWindow LocalWindow
}

func NewMonitorService(sms sms.Service, localWindow LocalWindow, MaxMissReq uint32, averageResp, longReq int64) MonitorServiceInterface {
	return &MonitorService{
		name:        "短信测试1",
		Service:     sms,
		status:      atomic.NewBool(true),
		percentage:  atomic.NewUint32(100),
		MaxMissReq:  MaxMissReq,
		AverageResp: averageResp,
		LongResp:    longReq,
		localWindow: localWindow,
	}
}

// Send

func (m *MonitorService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) (err error) {
	//TODO implement me
	//panic("implement me")
	var do bool
	do = m.Status()
	//这里当前m.Status() 是不会有 err 的
	if do {
		//正常设备接收流量
		//TODO 1全功率的健康设备
		if m.percentage.Load() >= 100 {
			err = m.doSend(ctx, biz, phoneNumbers, args, time.Now().Unix())
			return err
		}
		//TODO 2 异常变正常  需要慢慢预热,不扣分 交给定时任务去检查
		//随机数
		if uint32(rand.Int31n(101)) <= m.percentage.Load() {
			//这里不刷新m.CurrMissReq 是为了 尽快 把设备变成完全健康
			err = m.doSend(ctx, biz, phoneNumbers, args, time.Now().Unix())
			if err == nil {
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
			err = m.doSend(ctx, biz, phoneNumbers, args, time.Now().Unix())
			if err == nil {
				//正常处理需要加分 加 3 分
				m.percentage.Add(3)
			}
			return err

		}
		//太虚弱了  暂时不接收流量
		at.AddUint32(&m.CurrMissReq, 1)
		return ErrWeakToDo
	}

	//TODO 异常设备小流量去测试

	// 有一些不需要这样做 例如：使用监控他cpu\内存\io 之类的指标就行
	//
	//TODO 问题1:尽可能都把流量集中放到1个实例中,保证最快速度把 一个坏的实例 变成好的实例
	//必须有足够的测试样本才可以去检测
	// TODO 在满足flowReqToHandle的情况下,如果时间太长的话 滑动窗口 都过期了,检测就没有意义了
	// 所以需要只用一个时间窗口去统计样本
	err = m.doSend(ctx, biz, phoneNumbers, args, m.HappenErrTimeStamp)
	//if at.AddUint32(&m.OnErrReq, 1) >= flowReqToHandle {
	//	// 看看 有没有机会变好
	//	m.Statistics(m.HappenErrTimeStamp)
	//	at.StoreUint32(&m.OnErrReq, 0)
	//}
	if err != nil {
		// 前面应该跳过这个实例了 只要错了 就不给机会,留给下一个人 除非没有人可选
		at.StoreUint32(&m.OnErrReq, 0)
		at.StoreInt64(&m.HappenErrTimeStamp, time.Now().Unix())
	}
	return err
}

func (m *MonitorService) doSend(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal, timeStamp int64) (err error) {
	start := time.Now()
	err = m.Send(ctx, biz, phoneNumbers, args)
	costTime := time.Now().Sub(start).Milliseconds()
	m.Collect(MonitorCollect{
		err:        err,
		handleTime: costTime,
	}, timeStamp)
	return err
}

// Status 判断服务的状态
func (m *MonitorService) Status() bool {
	//可以放在Prometheus 然后通过 请求的方式 获取到指标数据 然后判断
	return m.status.Load()
}
func (m *MonitorService) SetStatus(status bool) {
	//可以放在Prometheus 然后通过 请求的方式 获取到指标数据 然后判断
	m.status.Store(status)
}

// Collect  收集服务的处理结果
func (m *MonitorService) Collect(mc MonitorCollect, timeStamp int64) {
	//判断MonitorCollect 的结构体然后晒到 收集器中
	m.localWindow.IncrementV1(mc, timeStamp, m.AverageResp, m.LongResp)
}

// Statistics 统计数据 判断实例是不是正常的
// 1.当前是本地的滑动窗口去处理
// 2.可以放在Prometheus 然后通过 请求的方式 获取到指标数据 然后判断
// 30的 检测限度可以参数传 或者 viper 动态监听去改就行
func (m *MonitorService) Statistics(timeStamp int64) (bool, error) {
	collect := m.localWindow.Statistics(timeStamp)
	//异常设备的检测次数过低 不做判断
	if uint32(collect.Count) < flowReqToHandle && !m.status.Load() {
		return false, ErrNoEnoughToCheck
	}
	//根据实际情况修改服务状态
	//m.status.Store(false)
	//这里写你的判断服务是否异常的标准
	//TODO 先把异常总数超过30% 就当做异常  这个指标值 应该弄一个结构体 让用户去传
	if (collect.OtherErr+collect.TimeOutErr/collect.Count)*100 >= 30 {
		//正常 -->异常
		//异常 -->继续异常
		m.status.Store(false)
		//重置异常发生时间
		at.StoreInt64(&m.HappenErrTimeStamp, time.Now().Unix())
		return false, nil
	} else {
		//异常 -->正常
		if m.status.CAS(false, true) {
			//设置流量承载能力
			m.percentage.Store(30)
			return true, nil
		}
		//正常 -->正常
		return true, nil
	}

}

func (m *MonitorService) GetName() string {
	return m.name
}

func (m *MonitorService) GetHappenErrTimeStamp() int64 {
	return at.LoadInt64(&m.HappenErrTimeStamp)
}

// Async  每个服务单独一个 还是这个层级 上移?
// 这里只会把好设备变成坏设备
func (m *MonitorService) Async() {

	go func() {
		ticker := time.NewTicker(m.interval)
		for {
			//这样写就不能控制了
			//<-ticker.C
			select {
			case <-ticker.C:

				healthy, err := m.Statistics(time.Now().Unix())
				if err != nil {
					fmt.Println("数据统计失败:", err)
					continue
				}
				m.status.Store(healthy)

			}
		}
	}()
}
