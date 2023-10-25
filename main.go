package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	glogger "gorm.io/gorm/logger"

	"gitee.com/geekbang/basic-go/webook/internal/repository/dao/article"
	local2 "gitee.com/geekbang/basic-go/webook/internal/service/sms/local"
	logger2 "gitee.com/geekbang/basic-go/webook/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	v9 "github.com/redis/go-redis/v9"

	"go.uber.org/zap"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache/local"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/internal/web"
	"gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/internal/web/middleware"
)

/*
k8s 打包
PS F:\git_push\webook>  go build -ldflags '-s -w' -tags="k8s" -o t99 .\main.go
测试环境打包
PS F:\git_push\webook>  go build -ldflags '-s -w' -o t99 .\main.go

*/

func main() {
	cfile := pflag.String("config", "F:\\git_push\\webook\\test_demo\\dev.yaml", "指定配置文件的路径")
	pflag.Parse()

	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	dsn := viper.GetString("db.mysql.dsn")
	fmt.Println("mysql 连接地址:", dsn)
	viper.WatchConfig()
	//TODO 数据库连接对象初始化
	logger := InitLogger()
	db, cache := initDB(logger)
	//gin 服务初始化
	l := logger2.NewNoOpLogger()
	jwtHandler := jwt.NewRedisJWTHandler(cache)
	user := initUser(db, cache, jwtHandler)
	a := initArticle(db, l)
	//中间件绑定以及路由注册
	server := initWebServer(jwtHandler, user, a)
	// 初始化 UserHandle

	server.Run(":8091")

}

//func main2() {
//	engine := InitWebServer()
//	engine.Run(":8787")
//}

// viper 测试
func mainInit() {
	//initViper()
	//initViperV1()
	//initViperV3()
	//ioc.InitMysql()
	//initLogger()
	//initLoggerv2()
	//日志初始化
	initLoggerv3()
}

func InitLogger() logger2.LoggerV1 {
	// 这里我们用一个小技巧，
	// 就是直接使用 zap 本身的配置结构体来处理
	cfg := zap.NewDevelopmentConfig()
	err := viper.UnmarshalKey("log", &cfg)
	if err != nil {
		panic(err)
	}
	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger2.NewZaplogger(l)
}

func initLogger() {
	//https://blog.csdn.net/LinAndCurry/article/details/122239544
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.L().Info("replace  前hello,go webook")
	//如果不replace  直接使用zap.L() 什么都打印不出来
	//替换全局zap 包变量
	zap.ReplaceGlobals(logger)

	zap.L().Info("hello,go webook")
	//Error  会打印堆栈信息
	zap.L().Error("验证码出错", zap.Error(errors.New("这是错误")))
	type A struct {
		Name string `json:"name1"`
	}
	zap.L().Info("验证码出错",
		zap.Error(errors.New("这是错误")),
		zap.Int16("id", 123),
		zap.Any("一个结构体", A{
			Name: "a",
		}))

}

// 适配器logger
func initLoggerv3() {
	//每个模块使用自定义的logger
	//这里有问题
	logger1, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	logger := logger2.NewZaplogger(logger1)
	logger.Info("掩码", logger2.String("phone", "13719088000"))

}

func initWebServer(jwtHandler jwt.Handler, userhandler *web.UserHandler, articleHandler *web.ArticleHandler) *gin.Engine {
	server := gin.Default()

	//TODO 通用中间件注册
	server.Use(func(ctx *gin.Context) {
		println("这是第一个 middleware")
	})

	server.Use(func(ctx *gin.Context) {
		println("这是第二个 middleware")
	})

	server.Use(cors.New(cors.Config{
		//AllowOrigins: []string{"*"},
		//AllowMethods: []string{"POST", "GET"},
		AllowHeaders:  []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{"x-jwt-token", "jwt-state", "x-refresh-token"},
		// 是否允许你带 cookie 之类的东西
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				// 你的开发环境
				return true
			}
			return strings.Contains(origin, "yourcompany.com")
		},
		MaxAge: 12 * time.Hour,
	}))
	//TODO session 配置
	//步骤1使用内存存储session
	//store := cookie.NewStore([]byte("secret"))

	//使用redis 存储session
	store, err := redis.NewStore(16, "tcp", config.Config.Redis.Addr, "",
		//自己生成的随机字符串
		[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"),
		[]byte("0Pf2r0wZBpXVXlQNdpwCXN4ncnlnZSc3"))
	if err != nil {
		panic(fmt.Sprintf("redis 连接失败:%v", err))
	}
	//给session  添加保存方式
	//表示前端的cookies 的Name 为 mysession
	server.Use(sessions.Sessions("mysession", store))
	// 步骤3
	//TODO  这是使用session  进行登录验证
	//server.Use(middleware.NewLoginMiddlewareBuilder().
	//	IgnorePaths("/users/signup").
	//	IgnorePaths("/users/login").Build())
	//TODO  这是使用JWT  进行登录验证

	//TODO base 上面的是通用中间件
	userhandler.RegisterPublicRoutes(server)
	articleHandler.RegisterPublicRoutes(server)

	//TODO 下面的是有私有中间件(要进行验证的)
	server.Use(middleware.NewLoginJWTMiddlewareBuilder(jwtHandler).
		//IgnorePaths("/users/signup").
		//IgnorePaths("/users/login_sms/code/send").
		//IgnorePaths("/users/login_sms").
		//IgnorePaths("/users/loginJWT").
		//IgnorePaths("/oauth2/wechat/authurl").IgnorePaths("/oauth2/wechat/callback").
		//IgnorePaths("/users/refresh_token").
		Build())

	userhandler.RegisterPrivateRoutes(server)
	articleHandler.RegisterPrivateRoutes(server)
	// v1
	//middleware.IgnorePaths = []string{"sss"}
	//server.Use(middleware.CheckLogin())

	// 不能忽略sss这条路径
	//server1 := gin.Default()
	//server1.Use(middleware.CheckLogin())
	return server
}

func initUser(db *gorm.DB, cache v9.Cmdable, jwtHandler jwt.Handler) *web.UserHandler {

	//user svc 构建
	ud := dao.NewUserDAO(db)
	uc := local.NewUserCache()
	repo := repository.NewUserRepository(ud, uc)
	svc := service.NewUserService(repo)

	//code svc 构建

	//阿里云短信服务
	//alSms := aliyun.NewAliyunService(
	//	"",
	//	"",
	//	"cn-hangzhou",
	//	"阿里云短信测试",
	//	"SMS_154950909",
	//)

	//redis短信验证服务
	//codeCache := cache2.NewRedisCodeCache(cache)
	//codeRepo := repository.NewCodeRepository(codeCache)

	//本地短信验证服务
	localCache := local.NewLocalSmsCache()
	codeRepo := repository.NewCodeRepository(localCache)
	localSms := local2.NewLocalSmsService()
	//使用本地短信(只打印出来验证码 不发短信 用于测试)
	codeSvc := service.NewCodeService(localSms, codeRepo)

	//使用阿里云 发送短信
	//codeSvc := service.NewCodeService(alSms, codeRepo)

	u := web.NewUserHandler(svc, codeSvc, jwtHandler)
	return u
}

func initArticle(db *gorm.DB,
	l logger2.LoggerV1) *web.ArticleHandler {
	gormDao := article.NewGORMArticleDAO(db)
	repo := repository.NewArticleRepository(gormDao, l)
	svc := service.NewArticleService(repo, l)
	return web.NewArticleHandler(svc, l)
}

func initDB(logger logger2.LoggerV1) (*gorm.DB, v9.Cmdable) {
	//mysql 自定义打印日志
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", config.Config.DB.DSN)), &gorm.Config{
		//需要一个logger glogger.New()
		Logger: glogger.New(gormLoggerFunc(logger.Debug), glogger.Config{
			//慢查询阈值
			SlowThreshold: time.Millisecond * 100,
			//是否忽略查找不到记录的异常
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			LogLevel:                  glogger.Info,
		}),
	})
	if err != nil {
		// 我只会在初始化过程中 panic
		// panic 相当于整个 goroutine 结束
		// 一旦初始化过程出错，应用就不要启动了
		panic(err)
	}
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	cache := v9.NewClient(&v9.Options{
		Addr:     config.Config.Redis.Addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	err = cache.Conn().Ping(ctx).Err()
	if err != nil {
		panic("redis  连接失败")
	}
	return db, cache
}

func initViperV3() {
	//TODO  启动命令
	//启动命令 go run .\main.go --config=./config/dev.yaml
	//默认值
	cfile := pflag.String("config", "./config/dev.yaml", "指定配置文件的路径")
	pflag.Parse()

	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func initViperReader() {
	viper.SetConfigType("yaml")
	cfg := `
db.mysql:
  dsn: "root:root@tcp(webook-mysql-service:3308)/webook"

redis:
  dsn: "webook-redis-service:6380"
`
	err := viper.ReadConfig(bytes.NewReader([]byte(cfg)))
	if err != nil {
		panic(err)
	}
}

func initViperV1() {
	viper.SetConfigFile("./config/dev.yaml")
	//这里可以设置默认值
	//viper.SetDefault("db.mysql.dsn","root:root@tcp(webook-mysql-service:3308)/webook")

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func initViper() {
	//配置文件的名字,但是不包含文件见扩展名
	viper.SetConfigName("dev")
	//配置文件的格式
	viper.SetConfigType("yaml")
	//当前文件的起点
	viper.AddConfigPath("./config")
	//可加多个
	//viper.AddConfigPath("")
	//viper.AddConfigPath("")

	//加载再内存里面(全局的)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	//可以生成一个新的viper 读取其他配置文件 然后把这个viper对象传给其他函数
	//otherViper := viper.New()
	//otherViper.SetConfigName("dev")
	////配置文件的格式
	//otherViper.SetConfigType("yaml")
	////当前文件的起点 可加多个
	//otherViper.AddConfigPath("./config")
	//otherViper.ReadInConfig()
}

// viper 连接etcd
func initViperRemote() {
	//endpoint 是集群的话 127.0.0.1:2379,127.0.0.3:2379,127.0.0.2:2379
	viper.SetConfigType("yaml")
	err := viper.AddRemoteProvider("etcd3", "127.0.0.1:2379", "/weebk")
	if err != nil {
		panic(err)
	}

	//Remote 是没有 OnConfigChange的使用了 即使用来也是没有反应的
	viper.OnConfigChange(func(in fsnotify.Event) {
		//只能知道变化了 但是 不知道那个数据发生变化了,只能重新读一次对应使用的配置
		//如
		//viper.GetString("mysql.dsn")
		fmt.Println(in.Name, in.Op)
	})

	err = viper.ReadRemoteConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(viper.AllKeys())
	fmt.Println(viper.AllSettings())
	//写到etcd 中
	//etcdctl --endpoints=127.0.0.1:2379 put /webook "$(<dev.yaml)"
}

// 监听配置文件变跟
func initViperWatchV1() {
	//TODO  启动命令
	//启动命令 go run .\main.go --config=./config/dev.yaml
	//默认值
	cfile := pflag.String("config", "./config/dev.yaml", "指定配置文件的路径")
	pflag.Parse()

	viper.SetConfigFile(*cfile)
	viper.WatchConfig()
	//开启之后 修改物理配置文件
	viper.OnConfigChange(func(in fsnotify.Event) {
		//只能知道变化了 但是 不知道那个数据发生变化了,只能重新读一次对应使用的配置
		//如
		//viper.GetString("mysql.dsn")
		fmt.Println(in.Name, in.Op)
	})

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

// 只有接口单方法能这样写
type gormLoggerFunc func(msg string, fields ...logger2.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, logger2.Field{
		Key:   "args",
		Value: args,
	})

}
