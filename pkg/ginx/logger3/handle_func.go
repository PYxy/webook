package logger3

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type Result struct {
	// 这个叫做业务错误码
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

var L logger.LoggerV1

// WrapReqForLogin 需要关注登录状态的
type WrapReqForLogin[req any] func(ctx *gin.Context, request req, uc *jwt.UserClaims) (Result, error)

func WrapReq[req any](fun WrapReqForLogin[req]) gin.HandlerFunc {

	return func(ctx *gin.Context) {
		var re req
		err := ctx.ShouldBind(&re)
		if err != nil {
			//bing  有异常绑定处理  直接返回就行
			fmt.Println("参数绑定异常:", err.Error())
			ctx.JSON(http.StatusOK, Result{
				Code: 0,
				Msg:  "参数合法性验证失败",
				Data: nil,
			})
			return
		}
		val, _ := ctx.Get("claims")
		uc, _ := val.(*jwt.UserClaims)
		res, err := fun(ctx, re, uc)
		if err != nil {
			fmt.Println("处理函数异常：", err)
			ctx.JSON(http.StatusOK, res)
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func WrapToken[C jwt.UserClaims](fn func(ctx *gin.Context, uc C) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 执行一些东西
		val, ok := ctx.Get("users")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c, ok := val.(C)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 下半段的业务逻辑从哪里来？
		// 我的业务逻辑有可能要操作 ctx
		// 你要读取 HTTP HEADER
		res, err := fn(ctx, c)
		if err != nil {
			// 开始处理 error，其实就是记录一下日志
			//L.Error("处理业务逻辑出错",
			//	logger.String("path", ctx.Request.URL.Path),
			//	// 命中的路由
			//	logger.String("route", ctx.FullPath()),
			//	logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
		// 再执行一些东西
	}
}
