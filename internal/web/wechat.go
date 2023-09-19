package web

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/lithammer/shortuuid/v4"

	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/internal/service/oauth2/wechat"
	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"

	"go.uber.org/atomic"
)

var stateKey atomic.String = atomic.String{}

//"95osj6fUD7foxmlYdDbncXz4VD2igvf1"

//微信认证流程
//1.请求服务的登录界面,选择微信登录
//2.后端服务构造请求微信登录的url 带上 回调uri、state 等信息给前端后端,前端自动跳转到微信扫码页面
//3.用户扫码 然后手机点击确认
//4.微信收到确认后, 带着access_token 请求 回调uri
//5.后端收到 微信返回的  access_token 跟state 之后进行验证
//6.验证通过之后,再次请求 微信 获取到长短token

//下面是错的
//注意 state 的作用
//由于state
//1.前端服务器 与 微信之间交互
//2.微信  与 后端服务进行交互
// state 不是现在在 扫码的uri 上面吗？

/*
微信登陆 state 的作用
微信响应回来的临时授权码 可以再浏览器中拦截到
1.攻击者先正常登陆 获取到临时授权码,正常登陆

2.攻击者 再页面上伪造一个点击按钮,攻击者带着用户的cookie(或者jwt token) 区请求,.攻击者的临时授权码(第一步的操作),请求绑定微信·,

******跟登陆没有关系*****

 拿着你的登陆状态 去绑定攻击者的微信账号
怎么拿呢
这就修改host 文件 把回调的域名映射到 本地
127.0.0.1   对应的域名   其实就是骗浏览器


结果: 攻击者 可以通过微信扫码登陆 成功看到用户的数据

噢 这里的关键是再获取回调url 的时候 把stae 放到 cookie(里面是jwt) 即使 能拿到临时授权码  也是跟这一次的请求回调url 生成的state 不一样
*/

type OAuth2WechatHandler struct {
	svc      wechat.Service
	userSvc  service.UserService
	stateKey []byte
	cfg      WechatHandlerConfig
	mJwt.Handler
}

type WechatHandlerConfig struct {
	Secure bool
	//StateKey
}

func NewOAuth2WechatHandler(svc wechat.Service, jwtHandle mJwt.Handler, cfg WechatHandlerConfig) *OAuth2WechatHandler {
	//stateKey 动态变法更配置的例子
	//
	stateKey.Store("123")
	viper.OnConfigChange(func(in fsnotify.Event) {
		stateKey.Store(viper.GetString("wechat.stateKey"))
	})
	return &OAuth2WechatHandler{
		svc: svc,
		//stateKey: []byte(stateKey),
		stateKey: []byte(stateKey.Load()),
		cfg:      cfg,
		Handler:  jwtHandle,
	}
}

func (h *OAuth2WechatHandler) RegisterRoutes(service *gin.Engine) {
	g := service.Group("/oauth2/wechat")
	g.GET("/authurl", h.AuthURL)
	g.Any("/callback", h.Callback)
}

func (h *OAuth2WechatHandler) AuthURL(ctx *gin.Context) {
	state := uuid.New()
	url, err := h.svc.AuthURL(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "构造扫码登录url 失败",
		})
		return
	}

	//设置jwtToken
	if err = h.setStateCookie(ctx, state); err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统异常",
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Data: url,
	})
}

func (h *OAuth2WechatHandler) setStateCookie(ctx *gin.Context, state string) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, StateClaims{
		State: state,
		RegisteredClaims: jwt.RegisteredClaims{
			// 过期时间，你预期中一个用户完成登录的时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
		},
	})
	//tokenStr, err := token.SignedString(h.stateKey)
	tokenStr, err := token.SignedString(stateKey.Load())

	if err != nil {
		return err
	}
	ctx.SetCookie("jwt-state", tokenStr,
		600, "/oauth2/wechat/callback",
		//线上环境    true            true  就很安全了
		"", h.cfg.Secure, true)
	return nil
}

func (h *OAuth2WechatHandler) Callback(ctx *gin.Context) {
	//获取临时授权码
	code := ctx.Query("code")
	//检查微信响应回来的state 是否更发送的一直
	//防止别人伪装
	err := h.verifyState(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "登录失败",
		})
		return
	}

	info, err := h.svc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}

	//拿到该微信用户的  OpenID  UnionID
	u, err := h.userSvc.CreatOrFindByWeChat(ctx, info)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}

	err = h.SetLoginToken(ctx, u.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Msg: "OK",
	})

}
func (h *OAuth2WechatHandler) verifyState(ctx *gin.Context) error {
	state := ctx.Query("state")
	// 校验一下我的 state
	ck, err := ctx.Cookie("jwt-state")
	if err != nil {
		return fmt.Errorf("拿不到 state 的 cookie, %w", err)
	}

	var sc StateClaims
	token, err := jwt.ParseWithClaims(ck, &sc, func(token *jwt.Token) (interface{}, error) {
		return h.stateKey, nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("token 已经过期了, %w", err)
	}

	if sc.State != state {
		return errors.New("state 不相等")
	}
	return nil
}

type StateClaims struct {
	State string
	jwt.RegisteredClaims
}
