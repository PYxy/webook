package myEtcd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"

	gresolver "google.golang.org/grpc/resolver"
)

var needToRegister = errors.New("需要重新绑定etcd")

type builder struct {
	c             *clientv3.Client
	endPoints     string
	r             *resolver
	currentFile   string
	checkInterval time.Duration
	em            *endpoints.Manager
}

func (b *builder) Build(target gresolver.Target, cc gresolver.ClientConn, opts gresolver.BuildOptions) (gresolver.Resolver, error) {
	fmt.Println("resolver Build")
	// Refer to https://github.com/grpc/grpc-go/blob/16d3df80f029f57cff5458f1d6da6aedbc23545d/clientconn.go#L1587-L1611
	endpoint := target.URL.Path
	if endpoint == "" {
		endpoint = target.URL.Opaque
	}
	endpoint = strings.TrimPrefix(endpoint, "/")
	b.r = &resolver{
		c:      b.c,
		target: endpoint,
		cc:     cc,
	}
	//r := &resolver{
	//	c:      b.c,
	//	target: endpoint,
	//	cc:     cc,
	//}
	//r.ctx, r.cancel = context.WithCancel(context.Background())
	//em, err := endpoints.NewManager(r.c, r.target)
	//if err != nil {
	//	return nil, status.Errorf(codes.InvalidArgument, "resolver: failed to new endpoint manager: %s", err)
	//}
	//r.wch, err = em.NewWatchChannel(r.ctx)
	//if err != nil {
	//	return nil, status.Errorf(codes.Internal, "resolver: failed to new watch channer: %s", err)
	//}
	//
	//r.wg.Add(1)
	//go r.watch()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if err := b.healthyCheck(ctx); err != nil {
		fmt.Println("客户端连接etcd 失败")
		panic(err)
	}
	fmt.Println("开搞、、、")
	time.Sleep(time.Second * 10)
	em, err := endpoints.NewManager(b.r.c, b.r.target)
	b.em = &em
	if err != nil {
		fmt.Println("客户端创建em 对象失败")
		//panic(err)
		//直接使用 默认的配置文件
		b.r.cc.UpdateState(gresolver.State{Addresses: b.LocalAddress()})
	}
	go b.syncTodo()
	return b.r, nil
}

func (b *builder) LocalAddress() []gresolver.Address {
	return []gresolver.Address{
		gresolver.Address{
			Addr: "127.0.0.1:8983",
			Metadata: map[string]any{
				"weight": 100,
				"cpu":    90,
				"vip":    "true",
				"ip":     "127.0.0.1:8983",
			},
		},
		gresolver.Address{
			Addr: "127.0.0.1:8982",
			Metadata: map[string]any{
				"weight": 100,
				"cpu":    90,
				"vip":    "true",
				"ip":     "127.0.0.1:8982",
			},
		},
	}
}
func (b *builder) healthyCheck(ctx context.Context) error {
	//endpoints 是逗号分隔的字符串  120.132.118.90:2379,120.132.118.90:2380,120.132.118.90:2379
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err := b.c.Status(ctx, b.endPoints)
	//fmt.Println("写操作:", err)
	return err
}

func (b *builder) ConnectToRA(ctx context.Context) error {
	//这里可以设置成 多次成功 增加检测间隔
	//失败 之后就马上通知服务端去注册
	//var times int
	var timer *time.Timer
	for {
		err := b.healthyCheck(ctx)
		if err != nil {
			//TODO  建议直接退出 ,不然续约那边断掉了 就起不来了
			return needToRegister
		}
		select {
		case <-ctx.Done():
			//服务端通知不需要检测了
			return ctx.Err()
		default:
			if timer == nil {
				timer = time.NewTimer(b.checkInterval)
			} else {
				timer.Reset(b.checkInterval)
			}
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}

		}

	}

}

func (b *builder) syncTodo() {
	var registerChan chan struct{}
	fmt.Println("开发")
	for {
		var onceClose sync.Once
		b.r.ctx, b.r.cancel = context.WithCancel(context.Background())
		registerChan = make(chan struct{}, 1)

		// 获取endPoint + 监控数据变化
		go func() {
			fmt.Println("????1")
			var tmpErr error
			b.r.wch, tmpErr = (*b.em).NewWatchChannel(b.r.ctx)
			if tmpErr != nil {
				onceClose.Do(func() {
					close(registerChan)
				})

			}
		}()
		//监控检查
		go func() {
			fmt.Println("????2")
			err := b.ConnectToRA(b.r.ctx)
			if errors.Is(err, needToRegister) {
				fmt.Println("健康检查协程 被强制退出(受影响与续约协程) 或 健康检查不通过")
				onceClose.Do(func() {
					close(registerChan)
				})
			} else {
				fmt.Println("健康检查协程正常退出")
			}

		}()
		time.Sleep(time.Second * 2)
		b.r.wg.Add(1)
		fmt.Println("?????3")
		go b.r.watch()
		select {
		case <-b.r.ctx.Done():
			fmt.Println("主动退出服务")
			return
		case <-registerChan:
			//加载默认的配置文件
			b.r.cancel()
			b.r.cc.UpdateState(gresolver.State{Addresses: b.LocalAddress()})
			fmt.Println("重启服务")

		}

	}

}

func (b *builder) Scheme() string {
	return "etcd"
}

// EtcdNewBuilder creates a resolver builder.
func EtcdNewBuilder(client *clientv3.Client, endPoints string, checkInterval time.Duration, currentFile string) (gresolver.Builder, error) {
	return &builder{
		c:             client,
		endPoints:     endPoints,
		currentFile:   currentFile,
		checkInterval: 0,
	}, nil
}

type resolver struct {
	c      *clientv3.Client
	target string
	cc     gresolver.ClientConn
	wch    endpoints.WatchChannel
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (r *resolver) watch() {
	defer r.wg.Done()
	defer func() {
		fmt.Println("watch 进程退出")
	}()
	allUps := make(map[string]*endpoints.Update)
	for {
		select {
		case <-r.ctx.Done():
			fmt.Println("??有问题")
			return
		case ups, ok := <-r.wch:
			if !ok {
				return
			}
			fmt.Println("ups", ups)

			for _, up := range ups {
				fmt.Println(up.Op)
				switch up.Op {
				case endpoints.Add:
					allUps[up.Key] = up
				case endpoints.Delete:
					delete(allUps, up.Key)
				}
			}
			fmt.Println("对象:", allUps)
			addrs := convertToGRPCAddress(allUps)
			for i := range addrs {
				fmt.Println(addrs[i].Metadata)
			}
			//手动添加节点进去
			addrs = append(addrs, gresolver.Address{
				Addr: "127.0.0.1:8983",
				Metadata: map[string]any{
					"weight": 100,
					"cpu":    90,
					"vip":    "true",
					"ip":     "127.0.0.1:8983",
				},
			})
			fmt.Println("更新可用节点信息:", addrs)
			r.cc.UpdateState(gresolver.State{Addresses: addrs})
		}
	}
}

func convertToGRPCAddress(ups map[string]*endpoints.Update) []gresolver.Address {
	var addrs []gresolver.Address
	for _, up := range ups {
		addr := gresolver.Address{
			Addr:     up.Endpoint.Addr,
			Metadata: up.Endpoint.Metadata,
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

// ResolveNow is a no-op here.
// It's just a hint, resolver can ignore this if it's not necessary.
func (r *resolver) ResolveNow(gresolver.ResolveNowOptions) {
	fmt.Println("resolver ResolveNow")

}

func (r *resolver) Close() {
	fmt.Println("调用了close方法。。。")
	r.cancel()
	r.wg.Wait()
}
