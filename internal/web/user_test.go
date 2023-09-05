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
	"gitee.com/geekbang/basic-go/webook/internal/service"
	svcmocks "gitee.com/geekbang/basic-go/webook/internal/service/mocks"
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
		mock     func(ctrl *gomock.Controller) (service.UserService, service.CodeService)
		reqBody  string
		wantCode int
		wantBody string
	}{

		{
			name: "正常注册",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(nil)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "注册成功",
		},
		{
			name: "参数绑定失败",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx",}`,
			wantCode: http.StatusBadRequest,
			wantBody: "",
		},
		{
			name: "邮箱格式有误",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "你的邮箱格式不对",
		},
		{
			name: "两次输入的密码不一致",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx1","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "两次输入的密码不一致",
		},
		{
			name: "密码必须大于8位，包含数字、特殊字符",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxxxxxxxxxxx","password":"xxxxxxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "密码必须大于8位，包含数字、特殊字符",
		},
		{
			name: "邮箱冲突",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(service.ErrUserDuplicate)
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
			},
			reqBody:  `{"email":"asxxxxxxxxxx@169.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"}`,
			wantCode: http.StatusOK,
			wantBody: "邮箱冲突",
		},
		{
			name: "系统异常",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "asxxxxxxxxxx@169.com",
					Password: "xxxx@1xxxxxxxxx",
				}).Return(errors.New("系统异常"))
				code := svcmocks.NewMockCodeService(ctrl)
				return user, code
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
		name      string
		mock      func(ctrl *gomock.Controller) (service.UserService, service.CodeService)
		reqBody   []Data
		wantCode  int
		wantBody  Result
		jwt_token string
	}{
		{
			name: "参数合法性验证失败",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {

				return nil, nil
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
			jwt_token: "",
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
			req.Header.Set("Content-Type", "application/json")

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
		})
	}
}

func TestUserHandler_LoginJWT(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctl *gomock.Controller) (service.UserService, service.CodeService)
		reqBody  string
		wantCode int
		wantBody string
		userId   int64 // jwt-token 中携带的信息
	}{
		{
			name: "正常登录",
			mock: func(ctrl *gomock.Controller) (service.UserService, service.CodeService) {
				user := svcmocks.NewMockUserService(ctrl)
				//写法1
				//user.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				//写法2  因为这个对象 是要跟request 对象的匹配上的
				user.EXPECT().Login(gomock.Any(), "asxxxxxxxxxx@169.com", "xxxx@1xxxxxxxxx").Return(
					domain.User{
						Id: 18,
					}, nil)
				code := svcmocks.NewMockCodeService(ctrl)

				return user, code
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
			tocker := resp.Header().Get("X-Jwt-Token")
			//t.Log(tocker)
			claims := &UserClaims{}
			token, err := jwt.ParseWithClaims(tocker, claims, func(token *jwt.Token) (interface{}, error) {
				//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
				return []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"), nil
			})
			if err != nil {
				t.Log(err)
				t.Fatal("解析jwttoken 失败")
			}
			t.Log(token)

			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			assert.Equal(t, tc.userId, claims.Uid)
		})
	}
}
