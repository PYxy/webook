package grpc

import (
	"errors"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"google.golang.org/grpc/resolver"
)

type nacosResolverBuilder struct {
	client naming_client.INamingClient
}

func (n *nacosResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	res := &nacosResolver{client: n.client, target: target, cc: cc}
	// 订阅服务器变更

	return res, res.subscribe()
}

func (n *nacosResolverBuilder) Scheme() string {
	return "nacos"
}

type nacosResolver struct {
	client naming_client.INamingClient
	target resolver.Target
	cc     resolver.ClientConn
}

func (r *nacosResolver) ResolveNow(options resolver.ResolveNowOptions) {
	// 注意，里面还有一个 SelectAllInstances，你要注意却别
	svcs, err := r.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: r.target.Endpoint(),
	})
	if err != nil {
		r.cc.ReportError(err)
		return
	}
	if len(svcs) == 0 {
		r.cc.ReportError(errors.New("无可用候选节点"))
		return
	}

}

func (r *nacosResolver) subscribe() error {
	err := r.client.Subscribe(&vo.SubscribeParam{
		ServiceName: r.target.Endpoint(),
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				// 记录日志就可以了，也就是服务发现中和注册中心出了问题
				return
			}
			err = r.reportAddrs(services)
			if err != nil {
				// 更新节点失败，一般也做不了什么，记录日志就可以
			}
		},
	})
	return err
}

func (r *nacosResolver) reportAddrs(svcs []model.Instance) error {
	addrs := make([]resolver.Address, 0, len(svcs))
	for _, svc := range svcs {
		addrs = append(addrs, resolver.Address{
			Addr: fmt.Sprintf("%s:%d", svc.Ip, svc.Port),
		})
	}
	return r.cc.UpdateState(resolver.State{
		Addresses: addrs,
	})
}

func (r *nacosResolver) Close() {
	// 不需要做啥
}
