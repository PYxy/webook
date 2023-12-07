package ioc

import (
	"go.uber.org/zap"

	//"gitee.com/geekbang/basic-go/webook/pkg/logger"

	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

func InitLogger() logger.LoggerV1 {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZaplogger(l)
}
