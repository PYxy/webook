package startup

import (
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

func InitLog() logger.LoggerV1 {
	return logger.NewNoOpLogger()
}
