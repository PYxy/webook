package middleware

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"
)

// LoginMiddlewareBuilder 扩展性
type LoginJWTMiddlewareBuilder struct {
	paths []string
	mJwt.Handler
}

func NewLoginJWTMiddlewareBuilder(jwtHandle mJwt.Handler) *LoginJWTMiddlewareBuilder {
	return &LoginJWTMiddlewareBuilder{
		Handler: jwtHandle,
	}
}
func (l *LoginJWTMiddlewareBuilder) IgnorePaths(path string) *LoginJWTMiddlewareBuilder {
	l.paths = append(l.paths, path)
	return l
}

func (l *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	// 用 Go 的方式编码解码

	gob.Register(time.Now())
	return func(ctx *gin.Context) {
		// 不需要登录校验的
		for _, path := range l.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}
		fmt.Println("?????????????????")
		//使用JWT 验证
		//要在postman 中的Headers 中添加Authorization 值为：bearer jwtTokenStr
		//tokenHeader := ctx.GetHeader("Authorization")

		//fmt.Println(tokenHeader)
		//if tokenHeader == "" {
		//	// 没登录
		//	ctx.AbortWithStatus(http.StatusUnauthorized)
		//	return
		//}
		//segs := strings.Split(tokenHeader, " ")
		//if len(segs) != 2 {
		//	//异常的Authorization 信息
		//	ctx.AbortWithStatus(http.StatusUnauthorized)
		//	return
		//}
		////获取实际的JWT tokenStr
		//tokenStr := segs[1]

		////解析到指定对象中
		//claims := &web.UserClaims{}
		//token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		//	//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
		//	return []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"), nil
		//})
		//if err != nil {
		//	ctx.AbortWithStatus(http.StatusUnauthorized)
		//	return
		//}
		//修改为长短token 之后
		tokenStr := l.ExtractToken(ctx)
		fmt.Println(tokenStr)
		//这里要更后面的强制类型转换对上
		claims := &mJwt.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
			return mJwt.AccessKey, nil
		})
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//检查过期时间
		if claims.ExpiresAt.Time.Before(time.Now()) {
			//过期了
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if token == nil || !token.Valid || claims.Uid == 0 {
			//解析成功  但是 token 以及 claims 不一定合法
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//可以增加一些二次验证(这么只是UserAgent)
		//App 可以增加一些 设备Id  mac 地址  运营商 之类的(短时间 不会发生变化的)
		if claims.UserAgent != ctx.Request.UserAgent() {
			//登录的UserAgent  跟现在的不一致
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//过期时间刷新
		//now := time.Now()
		//if now.Sub(claims.RegisteredClaims.ExpiresAt.Time) < time.Second*50 {
		//	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute * 10))
		//	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
		//	tokenStr, err = token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
		//	if err != nil {
		//		ctx.String(http.StatusInternalServerError, "系统错误")
		//		return
		//	}
		//	ctx.Header("x-jwt-token", tokenStr)
		//}
		fmt.Printf("检查claims: %#v \n", claims)
		err = l.CheckSession(ctx, claims.Ssid)
		if err == nil {
			//2.退出登录了
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//如果不等于nil
		if err != nil && err != redis.Nil {
			//redis 异常
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//这里放进去的是指针对象
		ctx.Set("claims", claims)
		//fmt.Println("火箭将获取到的对象是：", claims)
	}
}
