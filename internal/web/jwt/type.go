package jwt

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Handler interface {
	// SetLoginToken 登录成功之后设置 长短token
	SetLoginToken(ctx *gin.Context, uid int64) error
	// SetJWTToken 设置普通的jwtToken ssid的作用是判断登录状态
	SetJWTToken(ctx *gin.Context, uid int64, ssid string) error
	// ClearToken 退出登录的时候使用,清空前端的长短token  保存失效的ssid
	ClearToken(ctx *gin.Context) error
	// CheckSession 判断jwt 是否有效
	CheckSession(ctx *gin.Context, ssid string) error
	// ExtractToken 获取指定位置中的jwtToken
	ExtractToken(ctx *gin.Context) string
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int64
	Ssid string
	// 可以按需添加
	//UserAgent string
}

type UserClaims struct {
	jwt.RegisteredClaims
	// 声明你自己的要放进去 token 里面的数据
	Uid  int64
	Ssid string
	// 可以按需添加
	UserAgent string
}
