package wrr

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"go.uber.org/atomic"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const WeightRoundRobin = "weighted_round_robin"

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(WeightRoundRobin,
		&WeightedPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type WeightedPicker struct {
	mutex sync.Mutex
	conns []*weightConn
}

// Pick 对 balancer.Builder 获取到的后端实例进行筛选
func (b *WeightedPicker) Pickv2(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(b.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// 这里实时计算 totalWeight 是为了方便你作业动态调整权重
	var totalWeight int
	var res *weightConn

	b.mutex.Lock()
	for _, node := range b.conns {

		totalWeight += node.weight
		node.currentWeight += node.weight
		if res == nil || res.currentWeight < node.currentWeight {
			res = node
		}
	}
	res.currentWeight -= totalWeight
	b.mutex.Unlock()
	return balancer.PickResult{
		SubConn: res.SubConn,
		Done: func(info balancer.DoneInfo) {
			// 在这里执行 failover 有关的事情
			// 例如说把 res 的 currentWeight 进一步调低到一个非常低的值
			// 也可以直接把 res 从 b.conns 删除
			//不管是提高还是降低，都要设置一个阈值。比如说不允许降低到负数，不允许提高到某个极大的值。
			//并且思考，如果没有这个限制，可能发生什么。

			//最大值 +1 变成负数的最小值 开始 一个优先级极高的节点就会变成优先级极低的值,极大可能永远都不会选中他
			//+inf  0 +inf 应该都要关注一下
			b.mutex.Lock()
			defer b.mutex.Unlock()
			if info.Err == nil && res.currentWeight >= math.MaxInt {

				return
			}
			//如果是异常响应，也就是返回的 error 不为 nil，就降低权重。
			if info.Err != nil && res.currentWeight == 0 {

				return
			}

			if info.Err != nil {
				res.currentWeight--
			} else {
				res.currentWeight++
			}

		},
	}, nil
}

func (b *WeightedPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	fmt.Println("没请求一次都会进来一下这里 Pick")
	if len(b.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// 这里实时计算 totalWeight 是为了方便你作业动态调整权重
	var totalWeight int
	var res *weightConn

	b.mutex.Lock()
	for _, node := range b.conns {
		if !node.isActive.Load() {
			//熔断或者限流的 先排除
			continue
		}
		totalWeight += node.weight
		node.currentWeight += node.weight
		if res == nil || res.currentWeight < node.currentWeight {
			res = node
		}
	}
	res.currentWeight -= totalWeight
	b.mutex.Unlock()
	return balancer.PickResult{
		SubConn: res.SubConn,
		Done: func(info balancer.DoneInfo) {
			fmt.Println("响应的结果:", info.Err)
			// 在这里执行 failover 有关的事情
			// 例如说把 res 的 currentWeight 进一步调低到一个非常低的值
			// 也可以直接把 res 从 b.conns 删除
			//不管是提高还是降低，都要设置一个阈值。比如说不允许降低到负数，不允许提高到某个极大的值。
			//并且思考，如果没有这个限制，可能发生什么。

			//最大值 +1 变成负数的最小值 开始 一个优先级极高的节点就会变成优先级极低的值,极大可能永远都不会选中他
			//+inf  0 +inf 应该都要关注一下
			b.mutex.Lock()
			defer b.mutex.Unlock()

			if info.Err == nil {
				//正常响应
				if res.currentWeight >= math.MaxInt {
					//最大权重了
					return
				}
				//增加权重
				res.currentWeight++

			} else {
				// 熔断 || 限流
				//statu,_ := info.Err.
				isError, ok := info.Err.(interface{ Is(target error) bool })
				if !ok {
					fmt.Println("异常类型错误")
					return
				}
				//看下这样写是否正确
				fmt.Println(isError.Is(status.Error(codes.Unavailable, "不可用")))
				fmt.Println(isError.Is(status.Error(codes.ResourceExhausted, "不可用")))
				if isError.Is(status.Error(codes.Unavailable, "不可用")) || isError.Is(status.Error(codes.ResourceExhausted, "不可用")) {
					//响应状态异常
					fmt.Println("需要剔除")
					if res.currentWeight == 0 {
						//已经是最小权重了
						return
					}
					res.currentWeight--

					res.isActive.Store(false)
				}

			}

		},
	}, nil
}

type WeightedPickerBuilder struct {
}

// Build 获取etcd 中的所有后端信息 并针对metaData 中的数据进行整理 用于后面的Picker逻辑判断
func (b *WeightedPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	fmt.Println("BUILD-------------111")
	conns := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 如果不存在，那么权重就是 0
		weightVal, _ := conInfo.Address.Metadata.(map[string]any)["weight"]
		// 经过注册中心的转发之后，变成了 float64，要小心这个问题
		weight, _ := weightVal.(float64)
		conns = append(conns, &weightConn{
			SubConn:       con,
			isActive:      atomic.NewBool(true),
			weight:        int(weight),
			currentWeight: int(weight),
		})
	}
	//这样写测试不出来
	exitChan := make(chan struct{})
	runtime.SetFinalizer(b, func() {
		close(exitChan)
	})

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		defer func() {
			fmt.Println("检查协程退出....")
		}()
		for {
			select {
			case <-ticker.C:
				fmt.Println("定时任务....")
				//将异常的数据丢回去列表中
				for _, conn := range conns {
					if !conn.isActive.Load() {
						//进行必要的检查
						fmt.Println(conn.SubConn, "是异常的进行必要的检查")
						time.Sleep(time.Second * 2)
						//直接修改状态
						conn.isActive.Store(true)
					}
				}
			case <-exitChan:
				return
			}
		}
	}()
	return &WeightedPicker{
		conns: conns,
	}
}

type weightConn struct {
	// 初始权重
	weight int
	// 当前权重
	currentWeight int

	//针对 降级 熔断的时候进行限制
	isActive *atomic.Bool
	balancer.SubConn
}
