package wrr

import (
	"fmt"
	"math"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
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
func (b *WeightedPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
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

type WeightedPickerBuilder struct {
}

// Build 获取etcd 中的所有后端信息 并针对metaData 中的数据进行整理 用于后面的Picker逻辑判断
func (b *WeightedPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	fmt.Println("BUILD-------------")
	conns := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 如果不存在，那么权重就是 0
		weightVal, _ := conInfo.Address.Metadata.(map[string]any)["weight"]
		// 经过注册中心的转发之后，变成了 float64，要小心这个问题
		weight, _ := weightVal.(float64)
		conns = append(conns, &weightConn{
			SubConn:       con,
			weight:        int(weight),
			currentWeight: int(weight),
		})
	}
	return &WeightedPicker{
		conns: conns,
	}
}

type weightConn struct {
	// 初始权重
	weight int
	// 当前权重
	currentWeight int
	balancer.SubConn
}
