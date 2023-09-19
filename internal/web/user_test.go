package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/mock/gomock"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	svcmocks "gitee.com/geekbang/basic-go/webook/internal/service/mocks"
	mJwt "gitee.com/geekbang/basic-go/webook/internal/web/jwt"
)

// go test -v .

func TestEncrypt(t *testing.T) {
	password := "hello#world123"
	encrypted, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	err = bcrypt.CompareHashAndPassword(encrypted, []byte(password))
	assert.NoError(t, err)
}

func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler)
		reqBody  string
		wantCode int
		wantBody string
	}{

		{
			name: "正常注册",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(nil)
				code := svcmocks.NewMockCodeService(ctrl)
				//Mjwt :=jwtmocks.NewMockHandler(ctrl)
				//Mjwt.EXPECT().
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "注册成功",
		},
		{
			name: "参数绑定失败",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				//Mjwt :=jwtmocks.NewMockHandler(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx",}`,
			wantCode: http.StatusBadRequest,
			wantBody: "",
		},
		{
			name: "邮箱格式有误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "你的邮箱格式不对",
		},
		{
			name: "两次输入的密码不一致",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx1","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "两次输入的密码不一致",
		},
		{
			name: "密码必须大于8位，包含数字、特殊字符",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxxxxxxxxxxx","password":"xxxxxxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "密码必须大于8位，包含数字、特殊字符",
		},
		{
			name: "邮箱冲突",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(service.ErrUserDuplicate)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "邮箱冲突",
		},
		{
			name: "系统异常",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(errors.New("系统异常"))
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "系统异常",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			server := gin.New()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(server)

			req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)
			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersvc := svcmocks.NewMockUserService(ctrl)

	//Times(1)  预估运行多少次
	usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Times(1).
		Return(errors.New("mock error"))

	//usersvc.EXPECT().SignUp(gomock.Any(), domain.User{
	//	Email: "124@qq.com",
	//}).Return(errors.New("mock error"))

	err := usersvc.SignUp(context.Background(), domain.User{
		Email: "123@qq.com",
	})
	t.Log(err)
}

func TestUserHandler_SmsLogin(t *testing.T) {
	type Data struct {
		key   string
		value string
	}
	type Req struct {
		Phone string `form:"phone" binding:"required,len=11"` //电话号码是必须的,而且要11位
		Code  string `form:"code" binding:"required,len=6"`   //验证码 6位
	}

	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler)
		reqBody  []Data
		wantCode int
		wantBody Result
		userId   int64
	}{
		{
			name: "参数合法性验证失败",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {

				return nil, nil, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "12345",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "参数合法性验证失败",
			},
		},
		{
			name: "获取验证码之后  修改电话号码(搞事操作)",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				//uSvc :=svcmocks.NewMockUserService(ctrl)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(false, cache.ErrAttack)
				return nil, cSvc, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "请停止请求,或重新发送验证码登录",
			},
		},
		{
			name: "正常来说，如果频繁出现这个错误，你就要告警，因为有人搞你",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				//uSvc :=svcmocks.NewMockUserService(ctrl)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(false, cache.ErrCodeVerifyTooManyTimes)
				return nil, cSvc, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "连续输入错误验证码多次,请稍后再试(重新获取验证码)",
			},
		},
		{
			name: "未知错误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				//uSvc :=svcmocks.NewMockUserService(ctrl)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(false, cache.ErrUnknown)
				return nil, cSvc, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "未知错误",
			},
		},
		{
			name: "code 验证码有误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				//uSvc :=svcmocks.NewMockUserService(ctrl)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(false, nil)
				return nil, cSvc, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "验证码有误",
			},
		},
		{
			name: "usercache  系统异常",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				uSvc := svcmocks.NewMockUserService(ctrl)
				uSvc.EXPECT().CreateOrFind(gomock.Any(), "13719088020").Return(domain.User{}, service.ErrUnKnow)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(true, nil)
				return uSvc, cSvc, nil
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "系统异常",
			},
		},
		{
			name: "验证码校验通过,登录成功",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				uSvc := svcmocks.NewMockUserService(ctrl)
				uSvc.EXPECT().CreateOrFind(gomock.Any(), "13719088020").Return(domain.User{Id: 18}, nil)
				cSvc := svcmocks.NewMockCodeService(ctrl)
				cSvc.EXPECT().Verify(gomock.Any(), "login", "13719088020", "123456").Return(true, nil)
				//Mjwt := jwtmocks.NewMockHandler(ctrl)
				//Mjwt.EXPECT().SetLoginToken(gomock.Any(), int64(18)).Return(nil)
				Mjwt := mJwt.NewRedisJWTHandler(nil)
				return uSvc, cSvc, Mjwt
			},
			reqBody: []Data{
				{
					key:   "phone",
					value: "13719088020",
				},
				{
					key:   "code",
					value: "123456",
				},
			},
			wantCode: http.StatusOK,
			wantBody: Result{
				Code: 0,
				Msg:  "验证码校验通过,登录成功",
			},
			userId: 18,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			server := gin.New()
			h := NewUserHandler(tc.mock(ctl))
			h.RegisterRoutes(server)
			//
			DataUrlVal := url.Values{}
			for _, val := range tc.reqBody {
				DataUrlVal.Add(val.key, val.value)
			}
			req, err := http.NewRequest(http.MethodPost, "/users/login_sms", bytes.NewBuffer([]byte(DataUrlVal.Encode())))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			// json  字符串乱序的
			var resResult Result
			err = json.Unmarshal([]byte(resp.Body.String()), &resResult)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resResult)

			//解析JWT  token
			tocker := resp.Header().Get("X-Jwt-Token")
			if tocker != "" {
				//t.Log(tocker)
				claims := &mJwt.UserClaims{}
				_, err := jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
					//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
					return mJwt.AccessKey, nil
				})
				if err != nil {
					t.Log(err)
					t.Fatal("解析jwttoken 失败")
				}
				assert.Equal(t, tc.userId, claims.Uid)

			} else {
				t.Log("并没有登录成功,不需要验证token")
			}
			tocker = resp.Header().Get("X-Refresh-Token")
			if tocker != "" {
				//t.Log(tocker)
				claims := &mJwt.UserClaims{}
				_, err := jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
					//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
					return mJwt.RefreshKey, nil
				})
				if err != nil {
					t.Log(err)
					t.Fatal("解析jwttoken 失败")
				}
				assert.Equal(t, tc.userId, claims.Uid)

			} else {
				t.Log("并没有登录成功,不需要验证refresh token")
			}

		})
	}
}

func TestUserHandler_LoginJWT(t *testing.T) {
	testCases := []struct {
		name        string
		mock        func(ctl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler)
		reqBody     string
		wantCode    int
		wantBody    string
		userId      int64 // jwt-token 中携带的信息
		fingerprint string
	}{
		{
			name: "邮箱或密码不对",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				user.EXPECT().Login(gomock.Any(), "asxxxxxxxxxx@169.com", "xxxx@1xxxxxxxxx").Return(
					domain.User{}, service.ErrInvalidUserOrPassword)
				code := svcmocks.NewMockCodeService(ctrl)

				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "用户名或密码不对",
			userId:   0,
		},

		{
			name: "系统错误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				user.EXPECT().Login(gomock.Any(), "asxxxxxxxxxx@169.com", "xxxx@1xxxxxxxxx").Return(
					domain.User{}, service.ErrUnKnow)
				code := svcmocks.NewMockCodeService(ctrl)

				return user, code, nil
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "系统错误",
			userId:   0,
		},

		{
			name: "正常登录",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService, mJwt.Handler) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().Login(gomock.Any(), "asxxxxxxxxxx@169.com", "xxxx@1xxxxxxxxx").Return(
					domain.User{
						Id: 18,
					}, nil)
				code := svcmocks.NewMockCodeService(ctrl)

				Mjwt := mJwt.NewRedisJWTHandler(nil)
				return user, code, Mjwt
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "登录成功",
			userId:   18,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			server := gin.New()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(server)

			req, err := http.NewRequest(http.MethodPost, "/users/loginJWT", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			//用于接收resp
			resp := httptest.NewRecorder()
			server.ServeHTTP(resp, req)
			////获取响应头中的X-Jwt-Token 并解析
			//tocker := resp.Header().Get("X-Jwt-Token")
			////t.Log(tocker)
			//claims := &mJwt.UserClaims{}
			//token, err := jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
			//	//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
			//	return mJwt.AccessKey, nil
			//})
			//if err != nil {
			//	t.Log(err)
			//	t.Fatal("解析jwttoken 失败")
			//}
			//t.Log(token)
			assert.Equal(t, tc.wantBody, resp.Body.String())
			assert.Equal(t, tc.wantCode, resp.Code)

			if tc.wantBody == "登录成功" {

				//解析JWT  token
				tocker := resp.Header().Get("X-Jwt-Token")

				//t.Log(tocker)
				claims := &mJwt.UserClaims{}
				_, err := jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
					//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
					return mJwt.AccessKey, nil
				})
				if err != nil {
					t.Log(err)
					t.Fatal("解析jwttoken 失败")
				}
				assert.Equal(t, tc.userId, claims.Uid)

				tocker = resp.Header().Get("X-Refresh-Token")

				//t.Log(tocker)
				claims = &mJwt.UserClaims{}
				_, err = jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
					//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
					return mJwt.RefreshKey, nil
				})
				if err != nil {
					t.Log(err)
					t.Fatal("解析jwttoken 失败")
				}
				assert.Equal(t, tc.userId, claims.Uid)

			}

		})
	}
}

//
//func TestUserHandler_TokenLogin(t *testing.T) {
//	now := time.Now()
//	testCases := []struct {
//		name        string
//		mock        func(ctl *gomock.Controller) (service.UserService, service.CodeService)
//		reqBody     string
//		wantCode    int
//		wantBody    string
//		fingerprint string
//		//userId   int64 // jwt-token 中携带的信息
//	}{
//		{
//			name: "参数绑定失败",
//			mock: func(ctl *gomock.Controller) (service.UserService, service.CodeService) {
//				return nil, nil
//			},
//			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":"for-test"}`,
//			wantCode:    http.StatusBadRequest,
//			wantBody:    "参数合法性验证失败",
//			fingerprint: "",
//		},
//		{
//			name: "登录成功",
//			mock: func(ctl *gomock.Controller) (service.UserService, service.CodeService) {
//				return nil, nil
//			},
//			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
//			wantCode:    http.StatusOK,
//			wantBody:    "登陆成功",
//			fingerprint: "for-test",
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			ctrl := gomock.NewController(t)
//			defer ctrl.Finish()
//			server := gin.New()
//			h := NewUserHandler(tc.mock(ctrl))
//			h.RegisterRoutes(server)
//
//			req, err := http.NewRequest(http.MethodPost, "/users/login_token", bytes.NewBuffer([]byte(tc.reqBody)))
//			require.NoError(t, err)
//			req.Header.Set("Content-Type", "application/json")
//
//			//用于接收resp
//			resp := httptest.NewRecorder()
//
//			server.ServeHTTP(resp, req)
//
//			assert.Equal(t, tc.wantCode, resp.Code)
//
//			assert.Equal(t, tc.wantBody, resp.Body.String())
//			//登录成功才需要判断
//			if resp.Code == http.StatusOK {
//				access_token := resp.Header().Get("x-access-token")
//				refresh_token := resp.Header().Get("x-refresh-token")
//
//				//解析jwtToken
//				accessTokenClaim, err := UnPackJWT(access_token)
//				if err != nil {
//					panic(err)
//				}
//				//在判断 前端传入的信息是否一致
//				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
//				//判断过期时间
//				fmt.Println(now.Add(time.Minute * 29).UnixMilli())
//				fmt.Println(accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli())
//				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
//					panic("过期时间异常")
//					return
//				}
//
//				refreshTokenClaim, err := UnPackJWT(refresh_token)
//				if err != nil {
//					panic(err)
//				}
//				assert.Equal(t, tc.fingerprint, refreshTokenClaim.Fingerprint)
//				//判断过期时间
//				if now.Add(time.Hour*168).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
//					panic("过期时间异常")
//					return
//				}
//
//			}
//		})
//	}
//}

func UnPackJWT(tokenStr string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
		return []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"), nil
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		//解析成功  但是 token 以及 claims 不一定合法
		return nil, errors.New("不合法操作")
	}
	return claims, nil
}
