package job

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/atomic"
	"golang.org/x/sync/semaphore"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type Executor interface {
	// Executor 叫什么
	Name() string
	// Exec ctx 是整个任务调度的上下文
	// 当从 ctx.Done 有信号的时候，就需要考虑结束执行
	// 具体实现来控制
	// 真正去执行一个任务
	Exec(ctx context.Context, j domain.Job) error
}

type LocalFuncExecutor struct {
	funcs map[string]func(ctx context.Context, j domain.Job) error
	// fn func(ctx context.Context, j domain.Job)
}

func NewLocalFuncExecutor() *LocalFuncExecutor {
	return &LocalFuncExecutor{funcs: make(map[string]func(ctx context.Context, j domain.Job) error)}
}

func (l *LocalFuncExecutor) Name() string {
	return "local"
}

func (l *LocalFuncExecutor) RegisterFunc(name string, fn func(ctx context.Context, j domain.Job) error) {
	l.funcs[name] = fn
}

func (l *LocalFuncExecutor) Exec(ctx context.Context, j domain.Job) error {
	fn, ok := l.funcs[j.Name]
	if !ok {
		return fmt.Errorf("未知任务，你是否注册了？ %s", j.Name)
	}
	return fn(ctx, j)
}

// Scheduler 调度器
type Scheduler struct {
	execs   map[string]Executor
	svc     service.JobService
	l       logger.LoggerV1
	limiter *semaphore.Weighted

	name        string        //发布订阅的唯一标识
	targetChan  string        //发布订阅的通道
	myScore     *atomic.Int64 //自己的最低值
	OtherScore  *atomic.Int64 // 全部实例的最低值
	redisClient redis.Client
}

func NewScheduler(svc service.JobService, l logger.LoggerV1, name string, client redis.Client) *Scheduler {
	return &Scheduler{svc: svc, l: l,
		limiter: semaphore.NewWeighted(200),
		execs:   make(map[string]Executor),
		//-----------------
		name:        name,
		targetChan:  "jobToInteractive",
		myScore:     atomic.NewInt64(10),
		OtherScore:  atomic.NewInt64(0),
		redisClient: client,
	}

}

func (s *Scheduler) RegisterExecutor(exec Executor) {
	s.execs[exec.Name()] = exec
}

// 获取其他选手的负载
func (s *Scheduler) subscribe(ctx context.Context) {
	//
	//// 接收消息
	//for {
	//	msg, err := pubsub.ReceiveMessage(ctx)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	fmt.Println(msg.Channel, msg.Payload)
	//}
	pubsub := s.redisClient.Subscribe(ctx, s.targetChan)

	go func() {
		defer func() {
			_ = pubsub.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := pubsub.ReceiveMessage(ctx)
				if err != nil {
					fmt.Println(err)
					return
				}
				//fmt.Println(msg.Channel, msg.Payload)
				//这里不会有并发问题
				NewScore := gocast.Int64(strings.Split(msg.Payload, ":")[1])

				if s.OtherScore.Load() < NewScore {
					s.OtherScore.Store(NewScore)
				}
			}

		}
	}()

}

// 向其他选手发布自己的负载
func (s *Scheduler) publish(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				weight := rand.Int63n(100)
				s.myScore.Store(weight)
				err := s.redisClient.Publish(ctx, s.targetChan, fmt.Sprintf("%v:%v", s.name, weight)).Err()
				s.l.Error("发布自身负载失败", logger.Error(err))
				time.Sleep(time.Second * 2)
			}

		}
	}()
}

func (s *Scheduler) Schedule(ctx context.Context) error {
	for {

		if ctx.Err() != nil {
			// 退出调度循环
			return ctx.Err()
		}
		//分数比人家大就不抢了
		if s.myScore.Load() > s.OtherScore.Load() {
			continue
		}
		//负载均衡判断
		//随机数判断是不是负载超过多少就不错(实际按照任务的关键指标去判断)
		//要找一个中间体(mysql  redis 订阅功能 广播自己的负载)
		// 限制可同时运行的任务个数
		err := s.limiter.Acquire(ctx, 1)
		if err != nil {
			return err
		}
		// 一次调度的数据库查询时间
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		j, err := s.svc.Preempt(dbCtx)
		cancel()
		if err != nil {
			s.limiter.Release(1)
			// 你不能 return
			// 你要继续下一轮
			s.l.Error("抢占任务失败", logger.Error(err))
			continue
		}

		exec, ok := s.execs[j.Executor]
		if !ok {
			// DEBUG 的时候最好中断
			// 线上就继续
			s.l.Error("未找到对应的执行器",
				logger.String("executor", j.Executor))
			continue
		}

		// 接下来就是执行
		// 怎么执行？
		go func() {
			defer func() {
				// 释放
				s.limiter.Release(1)
				err1 := j.CancelFunc()
				if err1 != nil {
					s.l.Error("释放任务失败",
						logger.Error(err1),
						logger.Int64("jid", j.Id))
				}
			}()
			// 异步执行，不要阻塞主调度循环
			// 执行完毕之后
			// 这边要考虑超时控制，任务的超时控制
			err1 := exec.Exec(ctx, j)
			if err1 != nil {
				// 你也可以考虑在这里重试
				s.l.Error("任务执行失败", logger.Error(err1))
			}
			// 你要不要考虑下一次调度？
			if !s.svc.NeedToReset() {
				s.l.Error("协程续约失败,退出当前协程")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			//判断如果续约失败 这里就不要设置了(ResetNextTime 即使使用了版本去更新, 即使查不到去更新依旧返回nil)
			err1 = s.svc.ResetNextTime(ctx, j)
			if err1 != nil {
				s.l.Error("设置下一次执行时间失败", logger.Error(err1))
			}
		}()
	}
}
