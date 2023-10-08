package logger3

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"gitee.com/geekbang/basic-go/webook/internal/web/jwt"
)

type Result struct {
	// 这个叫做业务错误码
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

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
