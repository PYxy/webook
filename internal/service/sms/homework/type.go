package homework

import (
	"errors"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

var (
	// ErrWeakToDo 刚从异常变为健康状态 需要慢慢预热
	ErrWeakToDo = errors.New("虚弱状态")
)

// OPERATOR 根据当前服务李彪是不是存在异常设备 有就进行分流检查
type NewService interface {
	sms.Service
	Healthy() (bool, error)
	Statistics()
	Collect(mc MonitorCollect)
	GetName() string
}

type MonitorCollect struct {
	//监控字段 按需添加

	//
	//(超时异常 + 其他异常 + 连续超时 需要在获取的时候判断一下)
	err error
	//处理时间 (可用于 统计长尾 + 平均延时 + )
	handleTime int64
	//

}
