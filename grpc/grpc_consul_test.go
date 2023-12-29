package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConsulTestSuite struct {
	suite.Suite
	client *api.Client
}

func (s *ConsulTestSuite) SetupSuite() {
	cfg := api.DefaultConfig()
	cfg.Address = "120.132.118.90:8500"
	c, err := api.NewClient(cfg)
	require.NoError(s.T(), err)
	s.client = c
}

func (s *ConsulTestSuite) TestClient() {
	servicename := "user"
	cc, err := grpc.Dial(
		// consul服务
		fmt.Sprintf("consul://120.132.118.90:8500/%s?healthy=true", servicename),
		//"127.0.0.1:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	defer cc.Close()
	client := NewUserServiceClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//ctx = context.WithValue(ctx, "balancer-key", 123)
	resp, err := client.GetById(ctx, &GetByIdRequest{
		Id: 123,
	})
	require.NoError(s.T(), err)
	s.T().Log(resp.User)
	time.Sleep(time.Minute)

	return
	// 获取etcd 中所有的注册服务
	////servicename := "user"
	////serviceAddr := "127.0.0.1:8080"
	////id := fmt.Sprintf("%s-%s", servicename, serviceAddr)
	//fmt.Println(s.client.Agent().Services())
	//serviceMap, err := s.client.Agent().ServicesWithFilter("Service==`/service/user`")
	//if err != nil {
	//	fmt.Printf("query service from consul failed, err:%v\n", err)
	//	return
	//}
	//fmt.Println(serviceMap)
	//// 选一个服务机（这里选最后一个）
	//var addr string
	//for k, v := range serviceMap {
	//	fmt.Printf("%s:%#v\n", k, v)
	//	addr = v.Address + ":" + strconv.Itoa(v.Port)
	//	fmt.Println(addr)
	//}

}

func (s *ConsulTestSuite) TestServer() {
	l, err := net.Listen("tcp", ":8080")
	require.NoError(s.T(), err)
	//将grpc 服务器注册到 consul 上
	servicename := "user"
	addr := "127.0.0.1:8080"
	_ = &api.AgentServiceCheck{
		GRPC:                           addr,  // grpc 访问地址
		Timeout:                        "10s", // 超时时间
		Interval:                       "10s", // 健康检查频率
		DeregisterCriticalServiceAfter: "1m",  //1分钟后注销掉不健康的服务节点(最小1分钟)
	}

	agentService := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s", servicename, addr), // 服务唯一ID
		Name:    servicename,                             // 服务名称
		Tags:    []string{servicename, addr},             //服务标签
		Address: "127.0.0.1",
		Port:    8000,
		//Check:   check,
	}
	err = s.client.Agent().ServiceRegister(agentService)
	require.NoError(s.T(), err)

	server := grpc.NewServer()
	RegisterUserServiceServer(server, &Server{})
	fmt.Println("????")
	go func() {
		err = server.Serve(l)
		s.T().Log(err)
	}()
	time.Sleep(time.Second * 20)
	fmt.Println("服务强制停止....")
	// 注销服务
	err = s.client.Agent().ServiceDeregister(fmt.Sprintf("%s-%s", servicename, addr))
	s.T().Log(err)
	server.GracefulStop()
}

func TestConsul(t *testing.T) {
	suite.Run(t, new(ConsulTestSuite))
}
