//go:build wireinject

package main

import (
	"gitee.com/geekbang/basic-go/webook/pkg/wego"
	"gitee.com/geekbang/basic-go/webook/reward/grpc"
	"gitee.com/geekbang/basic-go/webook/reward/ioc"
	"gitee.com/geekbang/basic-go/webook/reward/repository"
	"gitee.com/geekbang/basic-go/webook/reward/repository/cache"
	"gitee.com/geekbang/basic-go/webook/reward/repository/dao"
	"gitee.com/geekbang/basic-go/webook/reward/service"
	"github.com/google/wire"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitRedis)

func Init() *wego.App {
	wire.Build(thirdPartySet,
		service.NewWechatNativeRewardService,
		ioc.InitAccountClient,
		ioc.InitGRPCxServer,
		ioc.InitPaymentClient,
		repository.NewRewardRepository,
		cache.NewRewardRedisCache,
		dao.NewRewardGORMDAO,
		grpc.NewRewardServiceServer,
		wire.Struct(new(wego.App), "GRPCServer"),
	)
	return new(wego.App)
}
