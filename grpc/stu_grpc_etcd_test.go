package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"gitee.com/geekbang/basic-go/webook/grpc/myEtcd"
	_ "gitee.com/geekbang/basic-go/webook/pkg/grpcx/balancer/pickOther" //匿名引入 用于自定义负载均衡策略
	_ "gitee.com/geekbang/basic-go/webook/pkg/grpcx/balancer/wrr"       //匿名引入 用于自定义负载均衡策略
)

type TestSuite struct {
	suite.Suite
	client *etcdv3.Client
}

func (s *TestSuite) SetupSuite() {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: []string{"120.132.118.90:2379"},
	})
	require.NoError(s.T(), err)
	s.client = client
}

func (s *TestSuite) TestClient2() {
	s.T()
	fmt.Println("--------------------------------------")
	bd, err := myEtcd.EtcdNewBuilder(s.client, "120.132.118.90:2379", time.Second*5, "./grpc.yaml")
	//bd, err := resolver.NewBuilder(s.client)
	require.NoError(s.T(), err)

	// URL 的规范 scheme:///xxxxx
	cc, err := grpc.Dial("etcd:///service/user",
		grpc.WithResolvers(bd),
		//grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"vip_pick":{}}]}`),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"my_round_robin":{}}]}`),

		//		grpc.WithDefaultServiceConfig(`{
		//"loadBalancingConfig": [{"round_robin":{}}]
		//}`),
		//grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"my_round_robin":{}}]}`),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			fmt.Println("中间件")
			err := invoker(ctx, method, req, reply, cc)
			if err != nil {
				fmt.Println(req)
				fmt.Println(reply)
				fmt.Println(method)
				fmt.Println(cc.GetState().String())
				fmt.Println("请求异常。。。:", err)
			}
			return err
		}),

		grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := NewUserServiceClient(cc)
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		//ctx = context.WithValue(ctx, "balancer-key", 123)
		resp, err := client.GetById(ctx, &GetByIdRequest{
			Id: 123,
		})
		require.NoError(s.T(), err)
		s.T().Log(resp.User)
		time.Sleep(time.Second * 3)
	}

}

func (s *TestSuite) TestClientWRR() {
	cfg := `
{
  "loadBalancingConfig": [{"weighted_round_robin":{}}],
  "methodConfig": [{
    "name": [{"service": "UserService"}],
    "retryPolicy": {
      "maxAttempts": 4,
      "initialBackoff": "0.01s",
      "maxBackoff": "0.1s",
      "backoffMultiplier": 2.0,
      "retryableStatusCodes": [ "UNAVAILABLE" ]
    }
  }]
}
`
	s.T()
	fmt.Println("--------------------------------------")
	bd, err := resolver.NewBuilder(s.client)
	require.NoError(s.T(), err)

	// URL 的规范 scheme:///xxxxx
	cc, err := grpc.Dial("etcd:///service/user",
		grpc.WithResolvers(bd),
		//grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"weighted_round_robin":{}}]}`),
		grpc.WithDefaultServiceConfig(cfg),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			fmt.Println("中间件")
			err := invoker(ctx, method, req, reply, cc)
			if err != nil {
				fmt.Println(req)
				fmt.Println(reply)
				fmt.Println(method)
				fmt.Println(cc.GetState().String())
				fmt.Println("请求异常。。。:", err)
			}
			return err
		}),

		grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := NewUserServiceClient(cc)
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		//ctx = context.WithValue(ctx, "balancer-key", 123)
		resp, err := client.GetById(ctx, &GetByIdRequest{
			Id: 123,
		})
		require.NoError(s.T(), err)
		s.T().Log(resp.User)
		time.Sleep(time.Second * 6)
	}
	//这是为了看下 健康检查协程会不会退出
	time.Sleep(time.Second * 5)

}

// 限流熔断 剔除节点然后恢复
func (s *TestSuite) TestServerFail() {
	go func() {
		s.ServiceFail("8983", "true")
	}()
	s.ServiceStart("8982", "true")
}

func (s *TestSuite) TestServer() {
	go func() {
		s.ServiceStart("8983", "true")
	}()
	s.ServiceStart("8982", "true")
}

func (s *TestSuite) ServiceFail(port, flag string) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}
	// endpoint 以服务为维度。一个服务一个 Manager
	em, err := endpoints.NewManager(s.client, "service/user")

	require.NoError(s.T(), err)
	addr := "127.0.0.1" + fmt.Sprintf(":%s", port)
	key := "service/user/" + addr

	var ttl int64 = 30
	ctxlease, cancel := context.WithTimeout(context.Background(), time.Second*2)
	leaseResp, err := s.client.Grant(ctxlease, ttl)
	require.NoError(s.T(), err)
	cancel()

	ctxAdd, cancel := context.WithTimeout(context.Background(), time.Second*2)
	err = em.AddEndpoint(ctxAdd, key, endpoints.Endpoint{
		Addr: addr,
		Metadata: map[string]any{
			"weight": 100,
			"cpu":    90,
			"vip":    flag,
		},
	}, etcdv3.WithLease(leaseResp.ID))

	cancel()
	//addr := netx.GetOutboundIP() + ":8090"
	// key 是指这个实例的 key
	// 如果有 instance id，用 instance id，如果没有，本机 IP + 端口
	// 端口一般是从配置文件里面读

	//... 在这一步之前完成所有的启动的准备工作，包括缓存预加载之类的事情
	kaCtx, kaCancel := context.WithCancel(context.Background())
	go func() {
		// 在这里操作续约
		ch, err1 := s.client.KeepAlive(kaCtx, leaseResp.ID)
		require.NoError(s.T(), err1)
		for kaResp := range ch {
			// 正常就是打印一下 DEBUG 日志啥的
			s.T().Log(kaResp.String(), time.Now().String())
		}
	}()
	server := grpc.NewServer()
	RegisterUserServiceServer(server, &ServerFail{Port: port})
	err = server.Serve(l)
	s.T().Log(err)

	// 你要退出了，正常退出
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 我要先取消续约
	kaCancel()
	// 退出阶段，先从注册中心里面删了自己
	err = em.DeleteEndpoint(ctx, key)

	// 关掉客户端
	s.client.Close()
	server.GracefulStop()
}
func (s *TestSuite) ServiceStart(port, flag string) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}
	// endpoint 以服务为维度。一个服务一个 Manager
	em, err := endpoints.NewManager(s.client, "service/user")

	require.NoError(s.T(), err)
	addr := "127.0.0.1" + fmt.Sprintf(":%s", port)
	key := "service/user/" + addr

	var ttl int64 = 30
	ctxlease, cancel := context.WithTimeout(context.Background(), time.Second*2)
	leaseResp, err := s.client.Grant(ctxlease, ttl)
	require.NoError(s.T(), err)
	cancel()

	ctxAdd, cancel := context.WithTimeout(context.Background(), time.Second*2)
	err = em.AddEndpoint(ctxAdd, key, endpoints.Endpoint{
		Addr: addr,
		Metadata: map[string]any{
			"weight": 100,
			"cpu":    90,
			"vip":    flag,
		},
	}, etcdv3.WithLease(leaseResp.ID))

	cancel()
	//addr := netx.GetOutboundIP() + ":8090"
	// key 是指这个实例的 key
	// 如果有 instance id，用 instance id，如果没有，本机 IP + 端口
	// 端口一般是从配置文件里面读

	//... 在这一步之前完成所有的启动的准备工作，包括缓存预加载之类的事情
	kaCtx, kaCancel := context.WithCancel(context.Background())
	go func() {
		// 在这里操作续约
		ch, err1 := s.client.KeepAlive(kaCtx, leaseResp.ID)
		require.NoError(s.T(), err1)
		for kaResp := range ch {
			// 正常就是打印一下 DEBUG 日志啥的
			s.T().Log(kaResp.String(), time.Now().String())
		}
	}()
	server := grpc.NewServer()
	RegisterUserServiceServer(server, &Server2{Port: port})
	err = server.Serve(l)
	s.T().Log(err)

	// 你要退出了，正常退出
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 我要先取消续约
	kaCancel()
	// 退出阶段，先从注册中心里面删了自己
	err = em.DeleteEndpoint(ctx, key)

	// 关掉客户端
	s.client.Close()
	server.GracefulStop()
}

func TestMyetcd(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
