package pickOther

import (
	"fmt"
	"math/rand"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

func init() {
	//以时间作为初始化种子
	rand.Seed(time.Now().UnixNano())
}

const VipPick = "vip_pick"

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(VipPick,
		&LabelPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type LabelPickerBuilder struct {
	c *clientv3.Client
}

func (l *LabelPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	fmt.Println("LabelPickerBuilder-------------")
	conns := make([]*labelConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 如果不存在，那么权重就是 0
		weightVal, _ := conInfo.Address.Metadata.(map[string]any)["weight"]
		flag, _ := conInfo.Address.Metadata.(map[string]any)["vip"]
		// 经过注册中心的转发之后，变成了 float64，要小心这个问题
		weight, _ := weightVal.(float64)
		vip, _ := flag.(string)
		conns = append(conns, &labelConn{
			weight:  int(weight),
			vip:     vip,
			SubConn: con,
		})
	}
	return &labelPicker{
		conns: conns,
	}
}

type labelConn struct {
	weight int
	vip    string
	time   time.Time
	balancer.SubConn
}

type labelPicker struct {
	conns []*labelConn
	next  int
}

// Pick  有并发风险 最好加个锁
func (l *labelPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	fmt.Println("=========labelPicker=============")
	if len(l.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	fmt.Println("可挑选的对象有:", len(l.conns))
	//获取前端的ctx 中的内容
	//ctx := info.Ctx
	//fmt.Println(ctx.Value("vip").(string))
	subConn := make([]*labelConn, 0, len(l.conns))
	for _, node := range l.conns {
		fmt.Println(node.vip)
		if node.vip == "true" {
			//fmt.Println("找到vip 节点,直接返回 或者 再进行复制均衡操作")
			//return balancer.PickResult{
			//	SubConn: node.SubConn,
			//}, nil
			subConn = append(subConn, node)
		}
	}
	if len(subConn) == 0 {
		//没找到vip 节点就随便挑一个
		fmt.Println("没找到vip 节点 随便返回一个")
		return balancer.PickResult{
			SubConn: l.conns[0].SubConn,
		}, nil
	}
	//随机返回一个
	node := subConn[l.next%len(subConn)]
	l.next += 1
	return balancer.PickResult{
		SubConn: node.SubConn,
		Done: func(info balancer.DoneInfo) {

			fmt.Println("响应结果:", info.Err)
			fmt.Println("响应结果:", info.Trailer.Get("ip"))
		},
	}, nil
}
