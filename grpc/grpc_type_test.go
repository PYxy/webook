package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
)

var needToRegister = errors.New("需要重新注册服务端")

type UserGrpcServer interface {
	Start(userService UserServiceServer, medaData map[string]any) error
	//RegisterServer grpc 服务注册
	RegisterServer(userService UserServiceServer)
	//RegisterToRA 向注册中心注册并保持续约
	RegisterToRA(ctx context.Context, medaData map[string]any)
	//UpdateToRA 更新 节点信息
	UpdateToRA(metaData map[string]any) error
	//Stop 关闭服务
	Stop()
}

type RegisterMidd interface {
	//RegisterToRA 注册 + 续约
	RegisterToRA(ctx context.Context, serviceKey, address string, medaData map[string]any)
	//UpdateToRA 更新 节点信息
	UpdateToRA(serviceKey, address string, metaData map[string]any) error
	//ConnectToRA 检查与注册中心之间的连接
	ConnectToRA(ctx context.Context) error

	DeleteEndpoint(serviceKey string) error
}

type EtcdRegisterMidd struct {
	client    *clientv3.Client
	endPoints string
	//用于检测的key  唯不唯一都可以
	Name string
	//续约间隔
	ttl int64
	//检测间隔
	checkInterval time.Duration
	//超时检测时间
	checkTimeOut time.Duration
	em           *endpoints.Manager
	leaseId      clientv3.LeaseID
	lock         sync.Mutex
	medaData     map[string]any
}

func NewEtcdRegisterMidd(
	endpoints string,
	name string, ttl int64,
	checkInterval time.Duration, checkTimeOut time.Duration,
	lock sync.Mutex) *EtcdRegisterMidd {
	//endpoints 是逗号分隔的字符串  120.132.118.90:2379,120.132.118.90:2380,120.132.118.90:2379
	endpointSeg := strings.Split(endpoints, ",")
	client, err := clientv3.New(clientv3.Config{
		//Endpoints: []string{"120.132.118.90:2379"},
		Endpoints: endpointSeg,
	})
	if err != nil {
		panic(err)
	}
	return &EtcdRegisterMidd{client: client, Name: name, ttl: ttl, checkInterval: checkInterval, checkTimeOut: checkTimeOut, lock: lock, endPoints: endpoints}

}

func (e *EtcdRegisterMidd) DeleteEndpoint(serviceKey string) error {
	fmt.Println("etcd 需要删除的key", serviceKey)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return (*e.em).DeleteEndpoint(ctx, serviceKey)
}

func (e *EtcdRegisterMidd) healthyCheck() error {
	//defer func() {
	//	err := recover()
	//	fmt.Println(err)
	//}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := e.client.Status(ctx, e.endPoints)
	//fmt.Println("写操作:", err)
	return err
}

func (e *EtcdRegisterMidd) RegisterToRA(ctxCtl context.Context, serviceKey, address string, medaData map[string]any) {
	//serviceKey  service/user
	//address 192.168.8.8:9090
	fmt.Println("RegisterToRA")
	fmt.Println("第一个:", serviceKey)
	em, err := endpoints.NewManager(e.client,
		serviceKey)
	if err != nil {
		panic(err)
	}
	e.em = &em
	for {
		fmt.Println("进来第一次")
		//需要在这里做一个锁  update 跟自动注册 加个锁  但是还有会有先后顺序的问题
		e.lock.Lock()
		if e.medaData == nil {
			e.medaData = medaData
		}
		medaData = e.medaData
		e.lock.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		// 例子以 service/user 为前缀
		//addr := fmt.Sprintf("%s:%d", m.Ip, m.Port)
		//key := m.serverKey + addr
		key := serviceKey + "/" + address
		leaseResp, err := e.client.Grant(ctx, e.ttl)
		//在这里还要再次判断 是不是正常设备
		err = e.healthyCheck()
		if err != nil {
			fmt.Println("注册后端服务之前,etcd 服务异常:", err)
			select {
			case <-ctxCtl.Done():
				fmt.Println("？？？？？？")
				return
			default:
				time.Sleep(time.Second * 3)
			}
			fmt.Println("????继续")
			continue
		}
		// metadata 一般用在客户端
		fmt.Println("第二个key:", key)
		err = em.AddEndpoint(ctx, key,
			endpoints.Endpoint{Addr: address, Metadata: medaData}, clientv3.WithLease(leaseResp.ID))
		if err != nil {
			//不退出 继续尝试,只打印日志
			//panic(err)
			fmt.Println("注册节点失败:", err)
			select {
			case <-ctxCtl.Done():
				return
			default:
				time.Sleep(time.Second * 5)
			}

			continue
		}
		e.leaseId = leaseResp.ID
		//异步任务
		cancelFunc, registerChan := e.SyncToDo()

		select {
		case <-ctxCtl.Done():
			fmt.Println("服务端主动要求退出")
			cancelFunc()
			//加个延时可以检查 异步协程有没有退出
			time.Sleep(time.Second * 3)
			return
		case <-registerChan:
			//需要帮服务端重新注册
			fmt.Println("重新开启异步任务")
			cancelFunc()
			continue
		}
	}

}

func (e *EtcdRegisterMidd) SyncToDo() (context.CancelFunc, chan struct{}) {
	keepCtx, keepCancel := context.WithCancel(context.Background())
	//续约
	go func() {
		ch, err := e.client.KeepAlive(keepCtx, e.leaseId)
		if err != nil {
			//应该尝试继续操作
			println("开启续约失败:", err)

		}
		for resp := range ch {
			//续约的日志
			println("续约日志:", resp.String())
		}
		fmt.Println("续约协程退出")
	}()
	var registerChan chan struct{}
	registerChan = make(chan struct{}, 1)
	//开始做健康检查
	go func() {
		err := e.ConnectToRA(keepCtx)
		if errors.Is(err, needToRegister) {
			close(registerChan)
		} else {
			fmt.Println("健康检查协程退出....")
			return
		}
	}()

	return keepCancel, registerChan
}

func (e *EtcdRegisterMidd) UpdateToRA(serviceKey, address string, metaData map[string]any) error {
	e.lock.Lock()
	e.medaData = metaData
	defer e.lock.Unlock()
	key := serviceKey + "/" + address
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// AddEndpoint 是一个覆盖的语义。也就是说，如果你这边已经有这个 key 了，就覆盖
	// upsert，set
	fmt.Println(key)
	err := (*e.em).AddEndpoint(ctx, key, endpoints.Endpoint{
		Addr: address,
		// 你们的分组信息，权重信息，机房信息
		// 以及动态判定负载的时候，可以把你的负载信息也写到这里
		//以下内容按需添加  参数自定义
		//Metadata: map[string]any{
		//	"weight": 200,
		//	"time":   now.String(),
		//},
		Metadata: metaData,
	}, clientv3.WithLease(e.leaseId))
	return err
}

func (e *EtcdRegisterMidd) ConnectToRA(ctx context.Context) error {
	//这里可以设置成 多次成功 增加检测间隔
	//失败 之后就马上通知服务端去注册
	//var times int
	for {

		err := e.healthyCheck()
		if err != nil {
			////如果是超时错误就给3次机会 可配置
			//if !errors.Is(err, context.DeadlineExceeded) {
			//	//操作失败
			//	fmt.Println("设置key 异常:", err)
			//	//通知外面要重新注册
			//	return needToRegister
			//}
			////超时错误就继续重试
			//if times >= 3 {
			//	//通知外面要重新注册
			//	return needToRegister
			//}
			////累计次数
			//times += 1

			//建议直接退出 ,不然续约那边断掉了 就起不来了
			return needToRegister
		}
		select {
		case <-ctx.Done():
			//服务端通知不需要检测了
			return ctx.Err()
		default:
			time.Sleep(e.checkInterval)
		}

	}

}

type MyUserServer struct {
	Ip        string
	Port      int
	regisMid  RegisterMidd
	serverKey string // "service/user"
	server    *grpc.Server
	cancelFun context.CancelFunc
}

func NewMyUserServer(ip string, port int, regisMid RegisterMidd, serverKey string) *MyUserServer {
	return &MyUserServer{Ip: ip, Port: port, regisMid: regisMid, serverKey: serverKey}
}

func (m *MyUserServer) RegisterServer(userService UserServiceServer) {
	m.server = grpc.NewServer()
	RegisterUserServiceServer(m.server, &Server{})
}

func (m *MyUserServer) RegisterToRA(ctx context.Context, medaData map[string]any) {
	m.regisMid.RegisterToRA(ctx, m.serverKey, fmt.Sprintf("%s:%d", m.Ip, m.Port), medaData)
}

func (m *MyUserServer) UpdateToRA(metaDate map[string]any) error {
	return m.regisMid.UpdateToRA(m.serverKey, fmt.Sprintf("%s:%d", m.Ip, m.Port), metaDate)

}

func (m *MyUserServer) Start(userService UserServiceServer, metaData map[string]any) error {
	address := fmt.Sprintf("%s:%d", m.Ip, m.Port)
	fmt.Println(address)
	listen, err := net.Listen("tcp", ":8091")
	if err != nil {
		panic(err)
	}

	m.RegisterServer(userService)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFun = cancel
	m.RegisterToRA(ctx, metaData)
	return m.server.Serve(listen)
}

func (m *MyUserServer) Stop() {
	//停止 注册中间的所有异步任务
	m.cancelFun()

	m.regisMid.DeleteEndpoint(fmt.Sprintf("%s/%s", m.serverKey, fmt.Sprintf("%s:%d", m.Ip, m.Port)))
	//grpc 服务优雅终止,给服务增加一些拦截以及一些 在处理服务的等待
	m.server.GracefulStop()

}

func Test_Server(t *testing.T) {

	regMid := NewEtcdRegisterMidd("120.132.118.90:2379", "text_server", 5, time.Second*2, time.Second*2, sync.Mutex{})
	server := NewMyUserServer("127.0.0.1", 8091, regMid, "service/user")

	//go func() {
	//	time.Sleep(time.Second * 20)
	//	server.Stop()
	//}()
	fmt.Println("服务启动、、、、、")
	fmt.Println(server.Start(&Server{Name: "内网"}, map[string]any{
		"weight": 200,
	}))
}

//第一个: service/user
//第一个: service/user
//第二个key: service/user/127.0.0.1:8091
//第二个key: service/user/127.0.0.1:8981
