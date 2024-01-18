package local_resolver

import (
	"fmt"

	gresolver "google.golang.org/grpc/resolver"

	ed "gitee.com/geekbang/basic-go/webook/grpc/myEtcd"
)

// 实现 resolver.Builder 接口
type LocalResolver struct {
}

func NewLocalResolver() *LocalResolver {
	return &LocalResolver{}

}

func LocalAddress(serviceName, filePath string) []gresolver.Address {

	ls := new(ed.LocalServer)
	if err := ed.ReadYaml(ls, filePath); err != nil {
		return []gresolver.Address{}
	}
	var resolverAddr []gresolver.Address
	serviceNode, ok := ls.GRPC[serviceName]
	if !ok {
		return []gresolver.Address{}
	}
	resolverAddr = make([]gresolver.Address, 0, len(serviceNode.Nodes))
	for _, node := range serviceNode.Nodes {
		resolverAddr = append(resolverAddr, gresolver.Address{
			Addr:       node.Address,
			ServerName: serviceName,
			Metadata:   node.Labels,
		})
	}
	return resolverAddr
}

func (l *LocalResolver) Build(target gresolver.Target, cc gresolver.ClientConn, opts gresolver.BuildOptions) (gresolver.Resolver, error) {

	cc.UpdateState(gresolver.State{Addresses: LocalAddress("service/user", "./grpc.yaml")})
	return &resolver{}, nil
}

func (l *LocalResolver) Scheme() string {
	return "local"
}

// 实现Resolver
type resolver struct {
}

func (r resolver) ResolveNow(options gresolver.ResolveNowOptions) {
	fmt.Println("ResolveNow")
}

func (r resolver) Close() {
	fmt.Println("clode")
}
