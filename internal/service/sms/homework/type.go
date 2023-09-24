package homework

import (
	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// MonitorServiceInterface 记录监控数据的装饰器
type MonitorServiceInterface interface {
	sms.Service
	//Status 状态获取
	Status() bool
	//SetStatus 状态设置
	SetStatus(bool)
	//Statistics 数据统计
	Statistics(timeStamp int64) (bool, error)
	//Collect 数据收集
	Collect(mc MonitorCollect, timeStamp int64)
	//GetName 获取服务标识 用于Prometheus 之类的数据查询
	GetName() string
	GetHappenErrTimeStamp() int64
}

// LocalWindow 检测数据保存的地方
type LocalWindow interface {
	IncrementV1(collect MonitorCollect, timeStamp int64, averageResp, longReq int64)
	RemoveOldBucketV1(timeStamp int64)
	Statistics(timeStamp int64) (collect Collect)
}

type AvailabilityInterface interface {
	sms.Service
	//AsyncToCheck 异步检测 服务列表的健康状态
	AsyncToCheck()
	// Async 任务失败重试
	Async()

	// SetOn 上线 可以用index 去查 感觉还是用标识容易记
	SetOn(string)
	// SetOff 下线 可以用index 去查 感觉还是用标识容易记
	SetOff(string)
	//AddService 增加服务
	AddService(MonitorServiceInterface)
}

type Strategy interface {
	// Next 返回下一次重试的间隔，如果不需要继续重试，那么第二参数返回 false
	Next() (time.Duration, bool)
}

type Random interface {
	Allow() bool
}

type MyRandom struct {
}

func (r *MyRandom) Allow() bool {
	if rand.Int31n(10) != 5 {
		return true
	}
	return false
}

// MonitorCollect ---- 按需添加字段就好
type MonitorCollect struct {
	//监控字段 按需添加
	//(超时异常 + 其他异常 + 连续超时 需要在获取的时候判断一下)
	err error
	//处理时间 (可用于 统计长尾 + 平均延时 + )
	handleTime int64
}

// Collect ---- 按需添加字段就好
type Collect struct {
	//基于超时的实时检测（连续超时）这里用超时百分比
	TimeOutErr int64
	//其他错误
	OtherErr int64
	//基于响应时间的实时检测（比如说，平均响应时间上升 20%）
	OverAverageResp int64
	//总次数
	Count int64
	//基于长尾请求的实时检测（比如说，响应时间超过 1s 的请求占比超过了 10%）
	LongResp int64
}
