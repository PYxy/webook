package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "gitee.com/geekbang/basic-go/webook/pkg/grpcx/balancer/wrr"
)

var signals = []os.Signal{
	os.Interrupt, os.Kill, syscall.SIGKILL,
	syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP,
	syscall.SIGABRT, syscall.SIGTERM,
}

type LoadBalancerTestSuite struct {
	suite.Suite
	// 借助 etcd 来做服务发现
	cli *clientv3.Client
}

func (s *LoadBalancerTestSuite) SetupSuite() {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"120.132.118.90:2379"},
	})
	require.NoError(s.T(), err)

	s.cli = client
}

func (s *LoadBalancerTestSuite) TestClientWeightedRoundRobin() {
	t := s.T()
	etcdResolver, err := resolver.NewBuilder(s.cli)
	assert.NoError(t, err)
	cc, err := grpc.Dial("etcd:///service/user",
		grpc.WithResolvers(etcdResolver),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"weighted_round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	userClient := NewUserServiceClient(cc)
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ctx = context.WithValue(ctx, "vip", "true")
		resp, err := userClient.GetById(ctx, &GetByIdRequest{
			Id: 123,
		})
		cancel()
		require.NoError(t, err)
		t.Log(resp.User)
	}
}

// TestServer 会启动两个服务器，一个监听 8090，一个监听 8091
func (s *LoadBalancerTestSuite) TestServer() {
	go func() {
		s.startWeightedServer(":8090", 10)
	}()
	s.startWeightedServer(":8091", 20)
}

func (s *LoadBalancerTestSuite) startWeightedServer(addr string, weight int) {
	fmt.Println("-----------------------")
	t := s.T()
	em, err := endpoints.NewManager(s.cli,
		"service/user")
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 要以 /service/user 为前缀
	addr = "127.0.0.1" + addr
	key := "service/user/" + addr
	// 5s
	var ttl int64 = 5
	leaseResp, err := s.cli.Grant(ctx, ttl)
	// metadata 一般用在客户端
	err = em.AddEndpoint(ctx, key,
		endpoints.Endpoint{
			Addr: addr,
			Metadata: map[string]any{
				"weight": weight,
			},
		}, clientv3.WithLease(leaseResp.ID))
	assert.NoError(t, err)

	// 忽略掉 ctx，因为在测试环境下，我们不需要手动控制退出续约
	kaCtx, _ := context.WithCancel(context.Background())
	go func() {
		_, err1 := s.cli.KeepAlive(kaCtx, leaseResp.ID)
		require.NoError(t, err1)
		//for resp := range ch {
		//	t.Log(resp.String())
		//}
	}()
	go func() {
		for {
			fmt.Println("这里有问题？？？？？？？？")
			ctx, calcel := context.WithTimeout(context.Background(), time.Second*3)
			fmt.Println(s.cli.Status(ctx, "120.132.118.90:2379"))
			calcel()
			time.Sleep(time.Second * 3)
		}
	}()
	server := grpc.NewServer()
	//优雅关闭服务
	RegisterUserServiceServer(server, &Server{Name: addr})
	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)

	server.Serve(l)
}

func TestLoadBalancer(t *testing.T) {
	suite.Run(t, new(LoadBalancerTestSuite))
}
