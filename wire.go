//go:build wireinject

package main

import (
	"github.com/gin-gonic/gin"

	"github.com/google/wire"

	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache/local"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	sla "gitee.com/geekbang/basic-go/webook/internal/service/sms/local"
	"gitee.com/geekbang/basic-go/webook/internal/web"
	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/ioc"
)

func InitWebServer() *gin.Engine {

	wire.Build(
		//数据库连接初始化
		//TODO  base
		ioc.InitMysql,
		//initDB,
		ioc.InitRedis,
		//redis 中间件
		ioc.InitRedisStore,

		//TODO dao
		dao.NewUserDAO,

		//TODO cache
		//法1 local cache

		local.NewLocalSmsCache, //用于保存,获取验证码
		local.NewUserCache,

		//法2 redis cache
		//rd.NewUserCache,
		//rd.NewRedisSmsCache,

		//TODO repo
		repository.NewUserRepository,
		repository.NewCodeRepository,

		//支撑CodeService
		//短信服务
		sla.NewLocalSmsService,
		//阿里云短信服务

		// TODO Code server
		service.NewCodeService,

		//TODO User server
		service.NewUserService,

		mJwt.NewRedisJWTHandler,

		//使用NewCodeService  NewUserService
		web.NewUserHandler,

		//TODO  中间件初始化
		ioc.InitMiddleWare,
		ioc.InitWeb,
	)

	return gin.Default()
}
