package ioc

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	intrv1 "gitee.com/geekbang/basic-go/webook/api/proto/gen/intr/v1"
	"gitee.com/geekbang/basic-go/webook/interactive/service"
	"gitee.com/geekbang/basic-go/webook/internal/web/client"
)

/*
db:
  dsn: "root:root@tcp(localhost:13316)/webook"

redis:
  addr: "localhost:6379"

abc: "helloabc" # v1
# abc: "helloabcdef" # v2

kafka:
  addrs:
    - "localhost:9094"

grpc:
  client:
    intr:
      addr: "localhost:8090"
      secure: false
      threshold: 100
*/

// InitIntrGRPCClient 替换 service.InteractiveService 就可以了 然后调用就全部使用对应的方法就行
func InitIntrGRPCClient(svc service.InteractiveService) intrv1.InteractiveServiceClient {
	type Config struct {
		Addr      string
		Secure    bool
		Threshold int32
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.intr", &cfg)
	if err != nil {
		panic(err)
	}
	var opts []grpc.DialOption
	if cfg.Secure {
		// 上面，要去加载你的证书之类的东西
		// 启用 HTTPS
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	cc, err := grpc.Dial(cfg.Addr, opts...)
	if err != nil {
		panic(err)
	}
	remote := intrv1.NewInteractiveServiceClient(cc)                  //初始化grpc client
	local := client.NewInteractiveServiceAdapter(svc)                 //初始化本地请求接口(使用装饰器的方式封装)
	res := client.NewGreyScaleInteractiveServiceClient(remote, local) //适配器
	// 我的习惯是在这里监听
	viper.OnConfigChange(func(in fsnotify.Event) {
		var cfg Config
		err = viper.UnmarshalKey("grpc.client.intr", &cfg)
		if err != nil {
			// 你可以输出日志
		}
		res.UpdateThreshold(cfg.Threshold)
	})
	return res
}
