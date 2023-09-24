package homework

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"go.uber.org/atomic"
	at "sync/atomic"
)

var (
	ErrToAsync     = errors.New("没找到合适的服务商,转异步中")
	overRetryTimes = errors.New("超过最大重试次数")
)

type Availability struct {
	//服务列表
	svcs []MonitorServiceInterface
	//可以用来记录服务上线情况
	state uint32
	//总服务个数
	length int64
	//存在有坏的设备的
	needToCheck atomic.Bool
	//检测时间间隔
	//这个时间 有点难预计 你要保证 异步发送 总共的时间不能大于这个
	interval time.Duration
	repo     repository.SmsRepository

	//startIndex int64 //默认是 -1
	retryFunc  func() Strategy
	randomFunc func() Random

	//用户记录当前异常设备的id
	errIdx int64
}

func NewAvailability(svcs []MonitorServiceInterface, interval time.Duration, repo repository.SmsRepository, retryFunc func() Strategy, randomFunc func() Random) AvailabilityInterface {
	av := &Availability{
		svcs:        svcs,
		state:       0,
		length:      int64(len(svcs)),
		needToCheck: atomic.Bool{},
		interval:    interval,
		repo:        repo,
		retryFunc:   retryFunc,
		randomFunc:  randomFunc,
	}
	for i := range svcs {
		av.state = av.state | (1 << i)
	}

	return av
}

func NewAvailabilityV1(svcs []MonitorServiceInterface, interval time.Duration, repo repository.SmsRepository, retryFunc func() Strategy, randomFunc func() Random) sms.Service {
	av := &Availability{
		svcs:        svcs,
		state:       0,
		length:      int64(len(svcs)),
		needToCheck: atomic.Bool{},
		interval:    interval,
		repo:        repo,
		retryFunc:   retryFunc,
		randomFunc:  randomFunc,
	}
	for i := range svcs {
		av.state = av.state | (1 << i)
	}
	av.Async()
	av.AsyncToCheck()
	return av
}

func (e *Availability) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO   可以一进来就直接先存在数据中,不然在跑的时候重启就会丢失数据,偷鸡先不做
	//是否进行了操作
	var (
		//  state  needToCheck 的状态获取未必是一致的 需要做个判断是不是有人做了
		send    bool
		sendErr error
		i       int64
	)
	//state := at.LoadUint32(&e.state)
	//如果是监控cpu 内存  磁盘网络io 等硬件资源是不需要这样做的
	//当前检测的是业务指标 需要放流量 或者 使用固定的测试流量也行
	//有多少概率用于异常实例检测 用户来判断
	//TODO  不满足几率 或者 没有异常的设备
	if !e.randomFunc().Allow() || !e.needToCheck.Load() {
		goto HealthToDO
	}
	//记录当前测试的索引
	if idx := at.LoadInt64(&e.errIdx); idx != 0 {
		sendErr = e.svcs[idx%at.LoadInt64(&e.length)].Send(ctx, biz, phoneNumbers, args)
		if sendErr != nil {
			//下次就不给他机会了
			// TODO 能换就换 不能换就是别人已经 动了
			/// TODO 这里累加可能会超过上线 或者 某个异常的设备一直坏 就一直 + 1
			//at.CompareAndSwapInt64(&e.errIdx, idx, idx+1)
			//还不如超之后直接不管 反正异常设备在 正常处理请求的时候 是只用一个时间窗口
			//                                异常才会创建一个新的窗口   对正常处理请求的 异常实例是没有影响的
			if at.AddInt64(&e.errIdx, 1) > 5 {
				at.StoreInt64(&e.errIdx, 0)
			}
			err := e.repo.Insert(ctx, domain.SMSBO{
				Biz:          biz,
				PhoneNumbers: phoneNumbers,
				Args:         args,
			})
			//cancel()
			if err != nil {
				fmt.Println("失败任务保存数据库失败:", err)
			}
		}

		return sendErr
	}
	for i = 0; i < at.LoadInt64(&e.length); i++ {
		//上线 且 非健康服务商
		if at.LoadUint32(&e.state)&(1<<i) != 0 && !e.svcs[i].Status() {
			//抢不到就算了
			at.CompareAndSwapInt64(&e.errIdx, 0, i)
			sendErr = e.svcs[i].Send(ctx, biz, phoneNumbers, args)
			send = true
			if sendErr == nil {
				//TODO 关注入口是不是保存到数据库了
				return sendErr
			}
			//ctx_, cancel := context.WithTimeout(context.Background(), time.Millisecond*400)
			err := e.repo.Insert(ctx, domain.SMSBO{
				Biz:          biz,
				PhoneNumbers: phoneNumbers,
				Args:         args,
			})
			//cancel()
			if err != nil {
				fmt.Println("失败任务保存数据库失败:", err)
			}
			return sendErr
			//}
		}
		//健康设备 那就idx 往前移一下
		at.CompareAndSwapInt64(&e.errIdx, i, 0)
	}

HealthToDO:
	for i = 0; i < at.LoadInt64(&e.errIdx); i++ {
		//TODO 这里有可能 正常设备 但是到后面处理的时候是异常设备,就变成 试探流量了
		// 上线且健康
		if at.LoadUint32(&e.state)&(1<<i) != 0 && e.svcs[i].Status() {
			//正常设备
			sendErr = e.svcs[i].Send(ctx, biz, phoneNumbers, args)
			send = true
			if sendErr == nil {

				return sendErr
			}
			if sendErr != nil && errors.Is(sendErr, ErrWeakToDo) {
				//刚恢复的设备
				send = false
				continue
			} else {
				send = true
				//正常设备处理请求,出现异常
				//ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*400)
				err := e.repo.Insert(ctx, domain.SMSBO{
					Biz:          biz,
					PhoneNumbers: phoneNumbers,
					Args:         args,
				})
				//cancel()
				if err != nil {
					fmt.Println("失败任务保存数据库失败:", err)
				}
				return sendErr
			}

		}
	}
	if !send {
		fmt.Println("没好到合适的设备发送, 转异步,请稍后..")
		//这里可以保存到数据库也行
		//ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*400)
		err := e.repo.Insert(ctx, domain.SMSBO{
			Biz:          biz,
			PhoneNumbers: phoneNumbers,
			Args:         args,
		})
		//cancel()
		if err != nil {
			fmt.Println("失败任务保存数据库失败:", err)
		}
		return ErrToAsync
	}

	return nil
}

// AsyncToCheck 服务列表健康检查只有这个地方能改变实例的状态
func (e *Availability) AsyncToCheck() {
	go func() {
		ticker := time.NewTicker(e.interval)
		//是不是有异常设备
		var (
			hasSick bool
			i       int64
		)
		for {
			select {
			case <-ticker.C:
				nowTimeStamp := time.Now().Unix()
				//state := at.LoadUint32(&e.state)
				for i = 0; i < at.LoadInt64(&e.length); i++ {
					//TODO 上线的 异常的设备
					if at.LoadInt64(&e.errIdx)&(1<<i) != 0 && !e.svcs[i].Status() {
						healthy, err := e.svcs[i].Statistics(e.svcs[i].GetHappenErrTimeStamp())
						//TODO 注意当前只会有一种错误
						if errors.Is(err, ErrNoEnoughToCheck) {
							hasSick = true
							fmt.Println(fmt.Sprintf("%s :异常状态(检测次数不足) %s", e.svcs[i].GetName(), err.Error()))
							continue
						}
						if !healthy {
							hasSick = true
						} else {
							//TODO 异常 -->正常
							//如果正在测试的异常设备是 当前索引 就要置0
							// TODO 能换就换 不能换就是别人已经 动了
							at.CompareAndSwapInt64(&e.errIdx, i, 0)
						}
						//TODO 异常 -->正常/异常
						e.svcs[i].SetStatus(healthy)
					}
					//TODO 上线的 正常的设备
					if at.LoadInt64(&e.errIdx)&(1<<i) != 0 && e.svcs[i].Status() {
						healthy, err := e.svcs[i].Statistics(nowTimeStamp)
						//TODO 这个位置不会有异常
						fmt.Println("上线的 正常的设备 数据统计出现异常:", err)
						if !healthy {
							//TODO 正常 -->异常
							hasSick = true
						}
						//TODO 正常 -->正常/异常
						e.svcs[i].SetStatus(healthy)
					}

				}

				e.needToCheck.Store(hasSick)
			}

		}

	}()
}

// Async 异常任务异步重试
func (e *Availability) Async() {
	go func() {
		ticker := time.NewTicker(e.interval)
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			//查询 任务创建时间超过多久的 就不要了或者直接全查
			msg, err := e.repo.Select(ctx, false, true)
			cancel()
			if err != nil {
				fmt.Println("想查询数据库失败")
			} else {
				//state := at.LoadUint32(&e.state)
				for _, singleMsg := range msg {
					strategy := e.retryFunc()
					//发送短信
					e.asyncSend(ctx, cancel, err, singleMsg, strategy)
				}
			}
		}
	}()
}

func (e *Availability) asyncSend(ctx context.Context, cancel context.CancelFunc, err error, singleMsg domain.SMSBO, strategy Strategy) {
	var (
		retryTimer *time.Timer
		i          int64
	)
	for i = 0; i < at.LoadInt64(&e.length); i++ {
		//TODO  感觉这里所有设备都可以重试
		//TODO  加速异常设备的测试
		//if state&(1<<i) != 0 {
		//ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)

		err = e.svcs[i].Send(ctx, singleMsg.Biz, singleMsg.PhoneNumbers, singleMsg.Args)
		//cancel()
		//第一次就成功
		if err == nil {
			//ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*400)
			err = e.repo.Update(ctx, singleMsg.Id, true, false)
			//cancel()
			if err != nil {
				fmt.Println(fmt.Sprintf("异步任务id:%v 重试成功,但是保存结果失败:%v", singleMsg.Id, err))
			}
			return
		}
		//发送失败
		timeInterval, try := strategy.Next()
		//超过重试次数
		if !try {
			fmt.Println(fmt.Sprintf("异步任务id:%v 重试失败:%v", singleMsg.Id, overRetryTimes))
			err = e.repo.Update(ctx, singleMsg.Id, false, false)
			if err != nil {
				fmt.Println(fmt.Sprintf("异步任务id:%v 更新数据库失败:%v", singleMsg.Id, err))
			}
			return
		}
		//还可以重试
		if retryTimer == nil {
			retryTimer = time.NewTimer(timeInterval)
		} else {
			retryTimer.Reset(timeInterval)
		}
		//重试的过程还要判断是不是已经超时了
		select {
		case <-ctx.Done():
			//超时直接退出
			fmt.Println(fmt.Sprintf("异步任务id:%v 重试过程出现超时异常:%v", singleMsg.Id, err))
			err = e.repo.Update(ctx, singleMsg.Id, false, false)
			if err != nil {
				fmt.Println(fmt.Sprintf("异步任务id:%v 更新数据库失败:%v", singleMsg.Id, err))
			}
			return

		case <-retryTimer.C:
			//继续下一个循环

		}
	}
}

// SetOn 上线
func (e *Availability) SetOn(target string) {
	for i := range e.svcs {
		if e.svcs[i].GetName() == target {
			at.StoreUint32(&e.state, at.LoadUint32(&e.state)-1<<i)
		}
	}
}

// SetOff 下线
func (e *Availability) SetOff(target string) {
	for i := range e.svcs {
		if e.svcs[i].GetName() == target {
			at.StoreUint32(&e.state, at.LoadUint32(&e.state)|(1<<i))
		}
	}
}

func (e *Availability) AddService(svc MonitorServiceInterface) {
	//TODO  顺序不能错
	//先加入切片
	e.svcs = append(e.svcs, svc)
	// 上线
	at.StoreUint32(&e.state, at.LoadUint32(&e.state)+(1<<at.LoadInt64(&e.length)))

	//增加长度
	at.AddInt64(&e.length, 1)
}
