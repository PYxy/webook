package grpc

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	gresolver "google.golang.org/grpc/resolver"
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

func (c *consulResolverBuilder) Build(target gresolver.Target, cc gresolver.ClientConn, opts gresolver.BuildOptions) (gresolver.Resolver, error) {
	fmt.Println("build1")
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
	cc          gresolver.ClientConn
	interval    time.Duration
}

func (c *consulResolver) watch() error {
	go func() {
		q := &api.QueryOptions{WaitIndex: 0}
		for {
			//services, meta, err := c.client.KV().Keys(c.target, "", q)
			services, meta, err := c.client.Health().Service(c.serviceName, c.serviceName, true, q)
			if err != nil {
				continue
			}
			fmt.Println(q.WaitIndex, meta.LastIndex)
			q.WaitIndex = meta.LastIndex

			var Addrs []gresolver.Address
			for _, service := range services {
				addr := fmt.Sprintf("%v:%v", service.Service.Address, service.Service.Port)
				Addrs = append(Addrs, gresolver.Address{
					Addr:       addr,
					ServerName: service.Service.Service,
				})
			}
			fmt.Println("获取到最新的addrs:", Addrs)
			err = c.cc.UpdateState(gresolver.State{Addresses: Addrs})
			fmt.Println("节点跟新结果:", err)
		}
	}()

	return nil
}

func (c *consulResolver) ResolveNow(options gresolver.ResolveNowOptions) {

}

func (c *consulResolver) Close() {
	//TODO implement me
	//panic("implement me")
}
