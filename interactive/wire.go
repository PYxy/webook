//go:build wireinject

package main

import (
	"github.com/google/wire"

	"gitee.com/geekbang/basic-go/webook/interactive/events"
	"gitee.com/geekbang/basic-go/webook/interactive/grpc"
	"gitee.com/geekbang/basic-go/webook/interactive/ioc"
	"gitee.com/geekbang/basic-go/webook/interactive/repository"
	"gitee.com/geekbang/basic-go/webook/interactive/repository/cache"
	"gitee.com/geekbang/basic-go/webook/interactive/repository/dao"
	"gitee.com/geekbang/basic-go/webook/interactive/service"
)

var thirdPartySet = wire.NewSet(ioc.InitDB,
	ioc.InitLogger,
	ioc.InitKafka,
	// 暂时不理会 consumer 怎么启动
	ioc.InitRedis)

var interactiveSvcProvider = wire.NewSet(
	service.NewInteractiveService,
	repository.NewCachedInteractiveRepository,
	dao.NewGORMInteractiveDAO,
	cache.NewRedisInteractiveCache,
)

func InitAPP() *App {
	wire.Build(interactiveSvcProvider,
		thirdPartySet,
		events.NewInteractiveReadEventConsumer,
		grpc.NewInteractiveServiceServer,
		ioc.NewConsumers,
		ioc.InitGRPCxServer,
		wire.Struct(new(App), "*"), //最终把所有的初始化对象结合之后 赋值到app 对象里面
	)
	return new(App)
}
