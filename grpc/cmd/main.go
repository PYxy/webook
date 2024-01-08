package main

import (
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"

	g1 "gitee.com/geekbang/basic-go/webook/grpc"
)

type ServiceUser struct {
	client *api.Client
}

func main() {
	l, err := net.Listen("tcp", ":8089")
	//将grpc 服务器注册到 consul 上
	servicename := "user"
	addr := "156.236.71.5:8089"
	//健康检查
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
		Address: "156.236.71.5",
		Port:    8089,
		//Check:   check, 这里不能加进去 因为是内网 不然consul 显示为异常
	}
	cfg := api.DefaultConfig()
	cfg.Address = "120.132.118.90:8500"
	c, err := api.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	s := ServiceUser{client: c}
	err = s.client.Agent().ServiceRegister(agentService)

	server := grpc.NewServer()
	g1.RegisterUserServiceServer(server, &g1.Server{Name: "外网"})
	fmt.Println("????")
	go func() {
		err = server.Serve(l)
		fmt.Println(err)
	}()
	time.Sleep(time.Hour * 60)
	fmt.Println("服务强制停止....")
	// 注销服务
	err = s.client.Agent().ServiceDeregister(fmt.Sprintf("%s-%s", servicename, addr))
	server.GracefulStop()
}
