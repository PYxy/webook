package web

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"gitee.com/geekbang/basic-go/webook/internal/service"
)

type UserHandleV2 struct {
	svc     service.UserService
	codeSvc service.CodeService
	Jwt     JWT
}

func NewUserHandleV2(svc service.UserService, codeSvc service.CodeService, jwt JWT) *UserHandleV2 {
	return &UserHandleV2{
		svc:     svc,
		codeSvc: codeSvc,
		Jwt:     jwt,
	}
}

func (u *UserHandleV2) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users2")
	//长短token
	ug.POST("/login_token", u.TokenLogin)
}

func (u *UserHandleV2) TokenLogin(ctx *gin.Context) {

	type TokenLoginReq struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		Fingerprint string `json:"fingerprint" binding:"required"` //你可以认为这是一个前端采集了用户的登录环境生成的一个码，你编码进去 JWT acccess_token 中。
	}
	var req TokenLoginReq
	err := ctx.ShouldBind(&req)
	if err != nil {
		//bing  有异常绑定处理  直接返回就行
		fmt.Println(err.Error())
		ctx.String(http.StatusBadRequest, "参数合法性验证失败")
		return
	}
	//验证登录用户合法性 获取个人信息查找的标识: 例如id
	tmpMap := map[string]string{
		//"id":id,
		"fingerprint": req.Fingerprint,
	}
	accessToken, refreshToken, err := u.Jwt.Encryption(tmpMap)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统异常")
		return
	}
	ctx.Header("x-access-token", accessToken)
	ctx.Header("x-refresh-token", refreshToken)
	//可以换一种方式保持到redis里面,避免refresh_token 被人拿到之后一直使用
	//可以使用MD5 转一下,或者直接截取指定长度的字符串 如: 以key 为 前面获取到的字符串
	ctx.String(http.StatusOK, "登陆成功")
}

//type TokenClaims struct {
//	jwt.RegisteredClaims
//	// 这是一个前端采集了用户的登录环境生成的一个码
//	Fingerprint string
//	//用于查找用户信息的一个字段
//	Id int64
//}
