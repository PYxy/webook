package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	v9 "github.com/redis/go-redis/v9"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache/local"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	lc "gitee.com/geekbang/basic-go/webook/internal/service/sms/local"
	"gitee.com/geekbang/basic-go/webook/internal/web"
	"gitee.com/geekbang/basic-go/webook/internal/web/middleware"
)

/*
k8s 打包
PS F:\git_push\webook>  go build -ldflags '-s -w' -tags="k8s" -o t99 .\main.go
测试环境打包
PS F:\git_push\webook>  go build -ldflags '-s -w' -o t99 .\main.go

*/

func main() {
	db, cache := initDB()
	server := initWebServer()

	u := initUser(db, cache)
	u.RegisterRoutes(server)

	server.Run(":8091")

}

func initWebServer() *gin.Engine {
	server := gin.Default()

	//TODO 中间件注册
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
		ExposeHeaders: []string{"x-jwt-token"},
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
	store := cookie.NewStore([]byte("secret"))

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
	server.Use(middleware.NewLoginJWTMiddlewareBuilder().
		IgnorePaths("/users/signup").
		IgnorePaths("/users/login_sms/code/send").IgnorePaths("/users/login_sms").
		IgnorePaths("/users/loginJWT").Build())
	// v1
	//middleware.IgnorePaths = []string{"sss"}
	//server.Use(middleware.CheckLogin())

	// 不能忽略sss这条路径
	//server1 := gin.Default()
	//server1.Use(middleware.CheckLogin())
	return server
}

func initUser(db *gorm.DB, cache v9.Cmdable) *web.UserHandler {

	ud := dao.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)

	localSms := &lc.Service{}
	//codeCache := cache2.NewRedisCodeCache(cache)
	//codeRepo := repository.NewCodeRepository(codeCache)
	localCache := local.NewLocalCache()
	codeRepo := repository.NewCodeRepository(localCache)
	codeSvc := service.NewCodeService(localSms, codeRepo)
	u := web.NewUserHandler(svc, codeSvc)
	return u
}

func initDB() (*gorm.DB, v9.Cmdable) {
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", config.Config.DB.DSN)))
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
