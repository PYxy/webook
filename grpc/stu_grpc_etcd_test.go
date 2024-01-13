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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"gitee.com/geekbang/basic-go/webook/grpc/myEtcd"
	_ "gitee.com/geekbang/basic-go/webook/pkg/grpcx/balancer/pickOther" //匿名引入 用于自定义负载均衡策略
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
	bd, err := myEtcd.EtcdNewBuilder(s.client, "120.132.118.90:2379", time.Second*5, "/etc/yisu")
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

func (s *TestSuite) TestServer() {
	go func() {
		s.ServiceStart("8983", "true")
	}()
	s.ServiceStart("8982", "true")
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
	//addr := netx.GetOutboundIP() + ":8090"
	// key 是指这个实例的 key
	// 如果有 instance id，用 instance id，如果没有，本机 IP + 端口
	// 端口一般是从配置文件里面读
	key := "service/user/" + addr
	//... 在这一步之前完成所有的启动的准备工作，包括缓存预加载之类的事情
	var kaCancel context.CancelFunc
	if port != "8984" {
		// 这个 ctx 是控制创建租约的超时
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		// ttl 是租期
		// 秒作为单位
		// 过了 1/3（还剩下 2/3 的时候）就续约
		var ttl int64 = 30
		leaseResp, err := s.client.Grant(ctx, ttl)
		require.NoError(s.T(), err)

		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		fmt.Println("第二个key:", key)
		err = em.AddEndpoint(ctx, key, endpoints.Endpoint{
			Addr: addr,
			Metadata: map[string]any{
				"weight": 100,
				"cpu":    90,
				"vip":    flag,
			},
		}, etcdv3.WithLease(leaseResp.ID))
		require.NoError(s.T(), err)
		var kaCtx context.Context
		kaCtx, kaCancel = context.WithCancel(context.Background())
		go func() {
			// 在这里操作续约
			ch, err1 := s.client.KeepAlive(kaCtx, leaseResp.ID)
			require.NoError(s.T(), err1)
			for kaResp := range ch {
				// 正常就是打印一下 DEBUG 日志啥的
				s.T().Log(kaResp.String(), time.Now().String())
			}
		}()
		go func() {
			ticker := time.NewTicker(time.Second * 5)
			// 万一，我的注册信息有变动，怎么办？
			n := 0
			for now := range ticker.C {
				ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
				// AddEndpoint 是一个覆盖的语义。也就是说，如果你这边已经有这个 key 了，就覆盖
				// upsert，set
				err = em.AddEndpoint(ctx1, key, endpoints.Endpoint{
					Addr: addr,
					// 你们的分组信息，权重信息，机房信息
					// 以及动态判定负载的时候，可以把你的负载信息也写到这里
					Metadata: map[string]any{
						"weight": 200 + n,
						"time":   now.String(),
						"vip":    "true",
					},
				}, etcdv3.WithLease(leaseResp.ID))
				if err != nil {
					s.T().Log(err)
				}
				n += 1
				cancel1()
			}
		}()
	}

	server := grpc.NewServer()
	RegisterUserServiceServer(server, &Server2{Port: port})
	err = server.Serve(l)
	s.T().Log(err)
	if port == "8982" {
		// 你要退出了，正常退出
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// 我要先取消续约
		kaCancel()
		// 退出阶段，先从注册中心里面删了自己
		err = em.DeleteEndpoint(ctx, key)
	}

	// 关掉客户端
	s.client.Close()
	server.GracefulStop()
}

func TestMyetcd(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
