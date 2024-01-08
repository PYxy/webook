package grpc

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"

	"google.golang.org/grpc/resolver"
)

type consulResolverBuilder struct {
	client      *api.Client
	interval    time.Duration
	serviceName string
}

func NewconsulResolverBuilder(client *api.Client, interval time.Duration, serviceName string) *consulResolverBuilder {

	return &consulResolverBuilder{
		client:      client,
		interval:    interval,
		serviceName: serviceName,
	}
}

func (c *consulResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	res := &consulResolver{
		client:      c.client,
		serviceName: c.serviceName,
		cc:          cc,
		interval:    c.interval,
	}

	// 订阅服务器变更

	return res, res.watch()
}

func (n *consulResolverBuilder) Scheme() string {
	return "consul"
}

type consulResolver struct {
	client      *api.Client
	serviceName string
	cc          resolver.ClientConn
	interval    time.Duration
}

func (c *consulResolver) watch() error {
	go func() {
		q := &api.QueryOptions{WaitIndex: 0}
		for {
			//services, meta, err := c.client.KV().Keys(c.target, "", q)
			services, meta, err := c.client.Health().Service(c.serviceName, c.serviceName, true, q)
			if err != nil {
				panic(err)
			}
			fmt.Println(q.WaitIndex, meta.LastIndex)
			q.WaitIndex = meta.LastIndex

			var Addrs []resolver.Address
			for _, service := range services {
				addr := fmt.Sprintf("%v:%v", service.Service.Address, service.Service.Port)
				Addrs = append(Addrs, resolver.Address{
					Addr:       addr,
					ServerName: service.Service.Service,
				})
			}
			fmt.Println("获取到最新的addrs:", Addrs)
			err = c.cc.UpdateState(resolver.State{Addresses: Addrs})
			fmt.Println("节点跟新结果:", err)
			time.Sleep(c.interval)

		}
	}()

	return nil
}

func (c *consulResolver) ResolveNow(options resolver.ResolveNowOptions) {

}

func (c *consulResolver) Close() {
	//TODO implement me
	panic("implement me")
}
