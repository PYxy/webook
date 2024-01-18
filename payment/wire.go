//go:build wireinject

package main

import (
	"gitee.com/geekbang/basic-go/webook/payment/grpc"
	"gitee.com/geekbang/basic-go/webook/payment/ioc"
	"gitee.com/geekbang/basic-go/webook/payment/repository"
	"gitee.com/geekbang/basic-go/webook/payment/repository/dao"
	"gitee.com/geekbang/basic-go/webook/payment/web"
	"gitee.com/geekbang/basic-go/webook/pkg/wego"
	"github.com/google/wire"
)

func InitApp() *wego.App {
	wire.Build(
		ioc.InitEtcdClient,
		ioc.InitKafka,
		ioc.InitProducer,
		ioc.InitWechatClient,
		dao.NewPaymentGORMDAO,
		ioc.InitDB,
		repository.NewPaymentRepository,
		grpc.NewWechatServiceServer,
		ioc.InitWechatNativeService,
		ioc.InitWechatConfig,
		ioc.InitWechatNotifyHandler,
		ioc.InitGRPCServer,
		web.NewWechatHandler,
		ioc.InitGinServer,
		ioc.InitLogger,
		wire.Struct(new(wego.App), "WebServer", "GRPCServer"))
	return new(wego.App)
}
