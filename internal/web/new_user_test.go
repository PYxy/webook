package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitee.com/geekbang/basic-go/webook/internal/service"
	jwtmocks "gitee.com/geekbang/basic-go/webook/internal/web/mocks"
)

func TestUserHandleV2_TokenLogin(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name        string
		mock        func(ctl *gomock.Controller) (service.UserService, service.CodeService, JWT)
		reqBody     string
		wantCode    int
		wantBody    string
		fingerprint string
		//userId   int64 // jwt-token 中携带的信息
	}{
		{
			name: "参数绑定失败",
			mock: func(ctl *gomock.Controller) (service.UserService, service.CodeService, JWT) {
				return nil, nil, nil
			},
			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    "参数合法性验证失败",
			fingerprint: "",
		},
		{
			name: "系统异常",
			mock: func(ctl *gomock.Controller) (service.UserService, service.CodeService, JWT) {
				jwt1 := jwtmocks.NewMockJWT(ctl)
				jwt1.EXPECT().Encryption(gomock.Any()).Return("", "", errors.New("系统异常"))
				return nil, nil, jwt1
			},
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusInternalServerError,
			wantBody:    "系统异常",
			fingerprint: "",
		},
		{
			name: "登录成功",
			mock: func(ctl *gomock.Controller) (service.UserService, service.CodeService, JWT) {
				jwt1 := jwtmocks.NewMockJWT(ctl)
				tokenStr, refreshToken := CreateToken()
				jwt1.EXPECT().Encryption(gomock.Any()).Return(
					tokenStr, refreshToken,
					nil)
				jwt1.EXPECT().Decrypt(gomock.Any()).Return(
					&TokenClaims{
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
						},
						Fingerprint: "for-test",
					}, nil,
				)
				jwt1.EXPECT().Decrypt(gomock.Any()).Return(
					&TokenClaims{
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour * 168)),
						},
						Fingerprint: "for-test",
					}, nil,
				)
				return nil, nil, jwt1
			},
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusOK,
			wantBody:    "登陆成功",
			fingerprint: "for-test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			server := gin.New()
			h := NewUserHandleV2(tc.mock(ctrl))
			h.RegisterRoutes(server)

			req, err := http.NewRequest(http.MethodPost, "/users2/login_token", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			//登录成功才需要判断
			if resp.Code == http.StatusOK {
				accessToken := resp.Header().Get("x-access-token")
				refreshToken := resp.Header().Get("x-refresh-token")
				acessT, err := h.Jwt.Decrypt(accessToken)
				if err != nil {
					panic(err)
				}
				accessTokenClaim := acessT.(*TokenClaims)
				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
				//判断过期时间
				fmt.Println(now.Add(time.Minute * 29).UnixMilli())
				fmt.Println(accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli())
				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
					return
				}

				refreshT, err := h.Jwt.Decrypt(refreshToken)
				if err != nil {
					panic(err)
				}
				refreshTokenClaim := refreshT.(*TokenClaims)
				assert.Equal(t, tc.fingerprint, refreshTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Hour*168).UnixMilli() < accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
					return
				}
			}
		})
	}
}

func CreateToken() (string, string) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: "for-test",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, _ := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))

	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 168))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, _ := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	return tokenStr, refreshToken
}

func TestJWT(t *testing.T) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: "for-test",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		fmt.Println(err)
	}

	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 168))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		//删除 access-token
		fmt.Println(err)
	}
	fmt.Println("短的:", tokenStr)
	fmt.Println("长的:", refreshToken)
}
