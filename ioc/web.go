package ioc

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/web"
	"gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/internal/web/middleware"
)

// InitWeb 服务创建
func InitWeb(middleWare []gin.HandlerFunc, handler *web.UserHandler) *gin.Engine {
	server := gin.Default()
	//路由注册
	handler.RegisterRoutes(server)

	//中间件注册
	server.Use(middleWare...)
	return server
}

// InitMiddleWare 中间件
func InitMiddleWare(store redis.Store, handler jwt.Handler) []gin.HandlerFunc {

	tmpMiddle := make([]gin.HandlerFunc, 0, 10)

	//常规中间件
	tmpMiddle = append(tmpMiddle, func(ctx *gin.Context) {
		println("这是第一个 middleware")
	})
	tmpMiddle = append(tmpMiddle, func(ctx *gin.Context) {
		println("这是第一个 middleware")
	})

	//跨域请求设置
	tmpMiddle = append(tmpMiddle, cors.New(cors.Config{
		//AllowOrigins: []string{"*"},
		//AllowMethods: []string{"POST", "GET"},
		// 运行接收 请求头的信息有那些
		AllowHeaders: []string{"Content-Type", "Authorization"},
		//运行你响应回去的头信息有那些
		ExposeHeaders: []string{"x-jwt-token"},
		// 允许你带 cookie 之类的东西
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

	//session  保存位置
	tmpMiddle = append(tmpMiddle, sessions.Sessions("mysession", store))

	//jwt Token 或者 session  需要过滤的url
	tmpMiddle = append(tmpMiddle, middleware.NewLoginJWTMiddlewareBuilder(handler).
		IgnorePaths("/users/signup").
		IgnorePaths("/users/login_sms/code/send").
		IgnorePaths("/users/login_sms").
		IgnorePaths("/users/loginJWT").
		IgnorePaths("/oauth2/wechat/authurl").IgnorePaths("/oauth2/wechat/callback").
		IgnorePaths("/users/refresh_token").Build())

	return tmpMiddle

}

// InitRedisStore 中间件支持
func InitRedisStore() redis.Store {
	store, err := redis.NewStore(16, "tcp", config.Config.Redis.Addr, "",
		//自己生成的随机字符串
		[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"),
		[]byte("0Pf2r0wZBpXVXlQNdpwCXN4ncnlnZSc3"))
	if err != nil {
		panic(fmt.Sprintf("redis store 连接失败:%v", err))
	}

	return store
}
