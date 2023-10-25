package ioc

import (
	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/internal/web"
	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"
)

// 不共享logger
func InitUserHander(svc service.UserService, codeSvc service.CodeService, jwtHandler mJwt.Handler) *web.UserHandler {
	// 日志对象创建
	//l, err := zap.NewDevelopment()
	//if err != nil {
	//	panic(err)
	//}
	//把这个l  传给这个初始化函数就好
	u := web.NewUserHandler(svc, codeSvc, jwtHandler)
	return u
}
