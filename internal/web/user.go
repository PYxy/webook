package web

import (
	"fmt"
	"net/http"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/pkg/ginx/logger3"
)

// UserHandler User服务路由定义
type UserHandler struct {
	svc         service.UserService
	codeSvc     service.CodeService
	emailExp    *regexp.Regexp
	passwordExp *regexp.Regexp
	mJwt.Handler
	valid *validator.Validate
}

// 假定 UserHandler 上实现了 handler 接口
var _ handler = (*UserHandler)(nil)

func NewUserHandler(svc service.UserService, codeSvc service.CodeService, jwtHandler mJwt.Handler) *UserHandler {
	const (
		emailRegexPattern    = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
		passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	)
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)
	return &UserHandler{
		svc:         svc,
		codeSvc:     codeSvc,
		Handler:     jwtHandler,
		emailExp:    emailExp,
		passwordExp: passwordExp,
		valid:       validator.New(),
	}
}

func (u *UserHandler) RegisterRoutesV1(ug *gin.RouterGroup) {
	ug.GET("/profile", u.Profile)
	ug.POST("/signup", u.SignUp)
	ug.POST("/login", u.Login)

	ug.POST("/edit", u.Edit)
}

func (u *UserHandler) RegisterPublicRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.POST("/signup", u.SignUp)
	ug.POST("/login", u.Login)
	ug.POST("/loginJWT", u.LoginJWT)
	ug.POST("/logoutJWT", u.LogoutJWT)

	//短信验证登录 并按实际情况注册
	ug.POST("/login_sms/code/send", u.SmsSend)
	ug.POST("/login_sms", u.SmsLogin)
	//长短token
	ug.POST("/login_token", u.TokenLogin)
	//更新长度token
	ug.POST("/refresh_token", u.RefreshToken)
}

func (u *UserHandler) RegisterPrivateRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.GET("/profile", u.Profile)
	ug.GET("/profileJWT", u.ProfileJWT)

	ug.POST("/edit", u.Edit)
	ug.POST("/edit2", u.Edit2)
	ug.POST("/edit3", logger3.WrapReq[EditReq](u.Edit3))

}

func (u *UserHandler) TokenLogin(ctx *gin.Context) {

	type TokenLoginReq struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		Fingerprint string `json:"fingerprint" binding:"required"` //你可以认为这是一个前端采集了用户的登录环境生成的一个码，你编码进去 EncryptionHandle acccess_token 中。
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
	err = u.setTokenJwt(ctx, req.Fingerprint)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统异常")
		return
	}
	ctx.String(http.StatusOK, "登陆成功")
}

func (u *UserHandler) setTokenJwt(ctx *gin.Context, fingerprint string) error {
	/*
		在登录成功的时候，返回两个 token，
		一个 access_token，一个 refresh_token。
		其中 access_token 被用来正常访问数据，
		    refresh_token 用来刷新 token。
		其中 access_token 被放到响应 header x-access-token 中，refresh token 被放到 x-refresh-token 中。
	*/
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: fingerprint,
	}
	//access_token 被用来正常访问数据
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	accessToken, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		return err
	}
	ctx.Header("x-access-token", accessToken)
	//refresh_token 用来刷新 access_token。
	//修改 refresh_token 的过期时间
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 168))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		//删除 access-token
		ctx.Writer.Header().Del("x-access-token")
		return err
	}
	//可以换一种方式保持到redis里面,避免refresh_token 被人拿到之后一直使用
	//可以使用MD5 转一下,或者直接截取指定长度的字符串 如: 以key 为 前面获取到的字符串
	ctx.Header("x-refresh-token", refreshToken)

	return nil
}

func (u *UserHandler) TokenEdit(ctx *gin.Context) {
	ctx.String(http.StatusOK, "允许操作")
}

func (u *UserHandler) SmsSend(ctx *gin.Context) {
	//获取前端的手机号码
	type SmsReq struct {
		Phone string `form:"phone" binding:"required,len=11"` //电话号码是必须的,而且要11位
	}
	var req SmsReq
	err := ctx.ShouldBind(&req)
	if err != nil {
		//bing  有异常绑定处理  直接返回就行
		fmt.Println(err.Error())
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  err.Error(),
		})
		return
	}
	//biz  可以写死
	// 如果需要验证的话  需要前端来传(就是后面的JWT 验证)
	err = u.codeSvc.Send(ctx, "login", req.Phone)
	switch err {
	case nil:
		ctx.JSON(http.StatusOK, Result{
			Code: 1,
			Msg:  "发送成功",
		})

	case service.ErrFrequentlyForSend:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			//Msg: err.Error(),
			Msg: "请求太频繁,请稍后重试",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			//Msg: err.Error(),
			Msg: "系统异常",
		})
	}

}

func (u *UserHandler) SmsLogin(ctx *gin.Context) {
	// 获取收集号码  以及验证码
	type Req struct {
		Phone string `form:"phone" binding:"required,len=11"` //电话号码是必须的,而且要11位
		Code  string `form:"code" binding:"required,len=6"`   //验证码 6位
	}
	var req Req
	err := ctx.ShouldBind(&req)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  "参数合法性验证失败",
		})
		return
	}

	ok, err := u.codeSvc.Verify(ctx, "login", req.Phone, req.Code)
	//if err != nil {
	//	fmt.Println(err)
	//	ctx.JSON(http.StatusOK, Result{
	//		Code: 0,
	//		Msg:  "系统异常",
	//	})
	//	return
	//}
	switch err {
	case service.ErrAttack:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			//Msg: err.Error(),
			//获取到验证码之后,电话输错了 不太可能
			Msg: "请停止请求,或重新发送验证码登录",
		})
		return
	case service.ErrCodeVerifyTooManyTimes:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			//Msg: err.Error(),
			Msg: "连续输入错误验证码多次,请稍后再试(重新获取验证码)",
		})
		return
	case service.ErrUnKnow:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			//Msg: err.Error(),
			Msg: "未知错误",
		})
		return
	}

	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  "验证码有误",
		})
		return
	}
	fmt.Println("验证通过")
	//手机号以及 验证码都输入正确
	//
	user, err := u.svc.CreateOrFind(ctx, req.Phone)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  "系统异常",
		})
		return
	}
	//保存jwt Token
	//fmt.Println(user.Id)
	//常规jwtToken
	//err = u.setJWTToken(ctx, user.Id)
	//Jwt 长短token
	err = u.SetLoginToken(ctx, user.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  "系统异常",
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "验证码校验通过,登录成功",
	})

}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	type SignUpReq struct {
		Email           string `json:"email"`
		ConfirmPassword string `json:"confirmPassword"`
		Password        string `json:"password"`
	}

	var req SignUpReq
	// Bind 方法会根据 Content-Type 来解析你的数据到 req 里面
	// 解析错了，就会直接写回一个 400 的错误
	if err := ctx.Bind(&req); err != nil {
		return
	}

	ok, err := u.emailExp.MatchString(req.Email)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "你的邮箱格式不对")
		return
	}
	if req.ConfirmPassword != req.Password {
		ctx.String(http.StatusOK, "两次输入的密码不一致")
		return
	}
	ok, err = u.passwordExp.MatchString(req.Password)
	if err != nil {
		// 记录日志
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "密码必须大于8位，包含数字、特殊字符")
		return
	}

	// 调用一下 svc 的方法
	err = u.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	if err == service.ErrUserDuplicate {
		ctx.String(http.StatusOK, "邮箱冲突")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统异常")
		return
	}

	ctx.String(http.StatusOK, "注册成功")
}

func (u *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 步骤2
	// 在这里登录成功了
	// 设置 session
	sess := sessions.Default(ctx)

	// 我可以随便设置值了
	// 你要放在 session 里面的值
	sess.Set("userId", user.Id)
	_ = sess.Save()
	ctx.String(http.StatusOK, "登录成功")
	return
}

func (u *UserHandler) LoginJWT(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err == service.ErrInvalidUserOrPassword {
		ctx.String(http.StatusOK, "用户名或密码不对")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 步骤2
	// 在这里登录成功了
	// 设置JWT  token
	//claims := LUserClaims{
	//	RegisteredClaims: jwt.RegisteredClaims{
	//		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	//	},
	//	Uid:       user.Id,
	//	UserAgent: ctx.Request.UserAgent(),
	//}
	//token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	//if err != nil {
	//	ctx.String(http.StatusInternalServerError, "系统错误")
	//	return
	//}
	//ctx.Header("x-jwt-token", tokenStr)

	err = u.SetLoginToken(ctx, user.Id)
	if err != nil {
		ctx.String(http.StatusOK, "系统异常")
	}

	ctx.String(http.StatusOK, "登录成功")
}

func (u *UserHandler) LogoutJWT(ctx *gin.Context) {
	err := u.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "退出登录失败",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "退出登录OK",
	})
}

func (u *UserHandler) LogOut(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	sess.Options(sessions.Options{
		MaxAge: -1,
	})
	_ = sess.Save()
	ctx.String(http.StatusOK, "退出成功")
}

func (u *UserHandler) Edit(ctx *gin.Context) {
	type EditReq struct {
		NickName string `form:"nick_name" validate:"omitempty,gte=3,lt=20" binding:"omitempty,gte=3,lt=20"`
		BirthDay string `form:"birthDay" validate:"omitempty,datetime=2006-01-02" binding:"omitempty,datetime=2006-01-02"`
		Describe string `form:"describe" validate:"omitempty,min=0,max=50" binding:"omitempty,min=0,max=50"`
	}
	var req EditReq
	// 使用binding 标签
	err := ctx.ShouldBind(&req)
	if err != nil {
		//bing  有异常绑定处理  直接返回就行
		fmt.Println(err.Error())
		ctx.String(http.StatusBadRequest, "参数合法性验证失败")
		return
	}
	//这要使用validate 标签
	//err = u.valid.Struct(req)
	//if err != nil {
	//	ctx.JSON(http.StatusBadRequest, "参数合法性验证失败")
	//	return
	//}

	//获取session信息 确认修改人是谁
	//旧版本 的获取userId
	//val, _ := ctx.Get("userId")

	//EncryptionHandle
	val, _ := ctx.Get("claims")
	claim, _ := val.(*LUserClaims)
	if err = u.svc.Edit(ctx, domain.User{
		//Id:       val.(int64),
		Id:       claim.Uid,
		NickName: req.NickName,
		BirthDay: req.BirthDay,
		Describe: req.Describe,
	}); err != nil {
		fmt.Println("数据修改失败:", err)
		ctx.String(http.StatusBadRequest, "修改失败")
	}

	ctx.String(http.StatusOK, "修改成功")
}

func (u *UserHandler) Edit2(ctx *gin.Context) {
	type EditReq struct {
		NickName string `form:"nick_name" validate:"omitempty,gte=3,lt=20" binding:"omitempty,gte=3,lt=20"`
		BirthDay string `form:"birthDay" binding:"omitempty"`
		Describe string `form:"describe" validate:"omitempty,min=0,max=50" binding:"omitempty,min=0,max=50"`
	}

	fun := func(ctx *gin.Context, editReq EditReq, uc *mJwt.UserClaims) (logger3.Result, error) {
		if err := u.svc.Edit(ctx, domain.User{
			//Id:       val.(int64),
			Id:       uc.Uid,
			NickName: editReq.NickName,
			BirthDay: editReq.BirthDay,
			Describe: editReq.Describe,
		}); err != nil {
			return logger3.Result{
				Code: 0,
				Msg:  "系统异常",
				Data: nil,
			}, err
		}

		//ctx.String(http.StatusOK, "修改成功")
		return logger3.Result{
			Code: 1,
			Msg:  "修改成功",
			Data: nil,
		}, nil
	}
	logger3.WrapReq[EditReq](fun)(ctx)
}

func (u *UserHandler) Edit3(ctx *gin.Context, req EditReq, uc *mJwt.UserClaims) (logger3.Result, error) {
	if err := u.svc.Edit(ctx, domain.User{
		//Id:       val.(int64),
		Id:       uc.Uid,
		NickName: req.NickName,
		BirthDay: req.BirthDay,
		Describe: req.Describe,
	}); err != nil {
		return logger3.Result{
			Code: 0,
			Msg:  "系统异常",
			Data: nil,
		}, err
	}

	return logger3.Result{
		Code: 1,
		Msg:  "修改成功",
		Data: nil,
	}, nil
}

func (u *UserHandler) Profile(ctx *gin.Context) {
	type EditReq struct {
		Email    string `json:"email"`
		NickName string `json:"nick_name"`
		BirthDay string `json:"birthDay"`
		Describe string `json:"describe"`
	}
	val, _ := ctx.Get("userId")
	user, err := u.svc.FindById(ctx, val.(int64))

	if err != nil {
		fmt.Println("数据获取失败:", err)
		ctx.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":     "0",
			"errmsg":   err.Error(),
			"userinfo": EditReq{},
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code":   "1",
		"errmsg": "获取成功",
		"userinfo": EditReq{
			Email:    user.Email,
			NickName: user.NickName,
			BirthDay: user.BirthDay,
			Describe: user.Describe,
		},
	})

	return
}

func (u *UserHandler) ProfileJWT(ctx *gin.Context) {
	type EditReq struct {
		Email    string `json:"email"`
		NickName string `json:"nick_name"`
		BirthDay string `json:"birthDay"`
		Describe string `json:"describe"`
	}
	val, _ := ctx.Get("claims")
	claims, _ := val.(*mJwt.UserClaims) // 这里要关注 前面初始化的实收是指针 还是 结构体
	fmt.Println(claims.Uid)
	user, err := u.svc.FindById(ctx, claims.Uid)

	if err != nil {
		fmt.Println("数据获取失败:", err)
		ctx.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":     "0",
			"errmsg":   err.Error(),
			"userinfo": EditReq{},
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code":   "1",
		"errmsg": "获取成功",
		"userinfo": EditReq{
			Email:    user.Email,
			NickName: user.NickName,
			BirthDay: user.BirthDay,
			Describe: user.Describe,
		},
	})

	return
}

func (u *UserHandler) setJWTToken(ctx *gin.Context, uid int64) error {
	claims := LUserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       uid,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误")
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

// RefreshToken 使用refresh token 更新access token
func (u *UserHandler) RefreshToken(ctx *gin.Context) {
	//先获取refresh_token
	refreshToken := u.ExtractToken(ctx)
	fmt.Println(refreshToken)
	var claims mJwt.RefreshClaims
	token, err := jwt.ParseWithClaims(refreshToken, &claims, func(token *jwt.Token) (interface{}, error) {
		//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
		return mJwt.RefreshKey, nil
	})
	fmt.Println(err, token)
	if err != nil || !token.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	//检查refresh token 是否已经(退出)
	err = u.CheckSession(ctx, claims.Ssid)
	//如果不等于nil
	if err != nil {
		//redis 异常
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	//创建一个新的access_token
	//TODO  感觉这里可以吧旧的access_token 也放在redis 中(redis  记录过期的的token  过期时间对上就行)
	// 不管也行 反正时间不长
	err = u.SetJWTToken(ctx, claims.Uid, claims.Ssid)
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "刷新成功",
	})
}

type LUserClaims struct {
	jwt.RegisteredClaims
	// 声明你自己的要放进去 token 里面的数据
	Uid int64
	// 自己随便加
	UserAgent string
}

type TokenClaims struct {
	jwt.RegisteredClaims
	// 这是一个前端采集了用户的登录环境生成的一个码
	Fingerprint string
	//用于查找用户信息的一个字段
	Id int64
}

type EditReq struct {
	NickName string `form:"nick_name" validate:"omitempty,gte=3,lt=20" binding:"omitempty,gte=3,lt=20"`
	BirthDay string `form:"birthDay" binding:"omitempty"`
	Describe string `form:"describe" validate:"omitempty,min=0,max=50" binding:"omitempty,min=0,max=50"`
}
