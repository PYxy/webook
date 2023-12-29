// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"gitee.com/geekbang/basic-go/webook/interactive/events"
	"gitee.com/geekbang/basic-go/webook/interactive/grpc"
	"gitee.com/geekbang/basic-go/webook/interactive/ioc"
	"gitee.com/geekbang/basic-go/webook/interactive/repository"
	"gitee.com/geekbang/basic-go/webook/interactive/repository/cache"
	"gitee.com/geekbang/basic-go/webook/interactive/repository/dao"
	"gitee.com/geekbang/basic-go/webook/interactive/service"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InitAPP() *App {
	loggerV1 := ioc.InitLogger()
	db := ioc.InitDB(loggerV1)
	interactiveDAO := dao.NewGORMInteractiveDAO(db)
	cmdable := ioc.InitRedis()
	interactiveCache := cache.NewRedisInteractiveCache(cmdable)
	interactiveRepository := repository.NewCachedInteractiveRepositoryv2(interactiveDAO, interactiveCache, loggerV1)
	interactiveService := service.NewInteractiveService(interactiveRepository, loggerV1)
	interactiveServiceServer := grpc.NewInteractiveServiceServer(interactiveService)
	server := ioc.InitGRPCxServer(interactiveServiceServer)
	client := ioc.InitKafka()
	interactiveReadEventConsumer := events.NewInteractiveReadEventConsumer(client, loggerV1, interactiveRepository)
	v := ioc.NewConsumers(interactiveReadEventConsumer)
	app := &App{
		server:    server,
		consumers: v,
	}
	return app
}

// wire.go:

var thirdPartySet = wire.NewSet(ioc.InitDB, ioc.InitLogger, ioc.InitKafka, ioc.InitRedis)

var interactiveSvcProvider = wire.NewSet(dao.NewGORMInteractiveDAO, cache.NewRedisInteractiveCache, repository.NewCachedInteractiveRepositoryv2, service.NewInteractiveService)