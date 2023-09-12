package web

import (
	"fmt"
	"net/http"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	jwt "github.com/golang-jwt/jwt/v5"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
)

// UserHandler User服务路由定义
type UserHandler struct {
	svc         service.UserService
	codeSvc     service.CodeService
	emailExp    *regexp.Regexp
	passwordExp *regexp.Regexp
	valid       *validator.Validate
}

// 假定 UserHandler 上实现了 handler 接口
//var _ handler = (*UserHandler)(nil)

func NewUserHandler(svc service.UserService, codeSvc service.CodeService) *UserHandler {
	const (
		emailRegexPattern    = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
		passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	)
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)
	return &UserHandler{
		svc:         svc,
		codeSvc:     codeSvc,
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

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.GET("/profile", u.Profile)
	ug.GET("/profileJWT", u.ProfileJWT)
	ug.POST("/signup", u.SignUp)
	ug.POST("/login", u.Login)
	ug.POST("/loginJWT", u.LoginJWT)
	ug.POST("/edit", u.Edit)

	//短信验证登录 并按实际情况注册
	ug.POST("/login_sms/code/send", u.SmsSend)
	ug.POST("/login_sms", u.SmsLogin)
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
	if err = u.setJWTToken(ctx, user.Id); err != nil {
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
	sess.Save()
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
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		Uid:       user.Id,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误")
		return
	}
	ctx.Header("x-jwt-token", tokenStr)
	ctx.String(http.StatusOK, "登录成功")
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

	//JWT
	val, _ := ctx.Get("claims")
	claim, _ := val.(*UserClaims)
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
	claims, _ := val.(*UserClaims)
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
	claims := UserClaims{
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

type UserClaims struct {
	jwt.RegisteredClaims
	// 声明你自己的要放进去 token 里面的数据
	Uid int64
	// 自己随便加
	UserAgent string
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
