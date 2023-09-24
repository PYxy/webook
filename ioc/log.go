package ioc

import "go.uber.org/zap"

func InitLogger() *zap.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	return l
}
