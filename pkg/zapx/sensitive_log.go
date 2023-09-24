package zapx

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type MyCore struct {
	zapcore.Core
}

func (c MyCore) Write(entry zapcore.Entry, fds []zapcore.Field) error {
	fmt.Println(entry)
	for _, fd := range fds {
		//掩码
		if fd.Key == "phone" {
			phone := fd.String
			//这个位置可能消耗很多性能
			fd.String = phone[:3] + "****" + phone[7:]
		}
	}
	return c.Core.Write(entry, fds)
}

func Mask(key string, val string) zap.Field {
	//注意长度问题
	val = val[:3] + "****" + val[7:]
	return zap.Field{
		Key:       key,
		String:    val,
		Interface: nil,
	}
}
