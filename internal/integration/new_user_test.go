package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitee.com/geekbang/basic-go/webook/internal/web"
)

func TestUserHandleV2_TokenLogin(t *testing.T) {
	server := gin.Default()
	userHandle := web.UserHandleV2{Jwt: web.NewJwt()}
	userHandle.RegisterRoutes(server)
	now := time.Now()
	testCases := []struct {
		name        string
		reqBody     string
		wantCode    int
		wantBody    string
		fingerprint string
		//userId   int64 // jwt-token 中携带的信息
	}{
		{
			name:        "参数绑定失败",
			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":""}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    "参数合法性验证失败",
			fingerprint: "",
		},
		{
			name:        "登陆成功",
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"long-short-token"}`,
			wantCode:    http.StatusOK,
			wantBody:    "登陆成功",
			fingerprint: "long-short-token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//构造请求
			req, err := http.NewRequest(http.MethodPost, "/users2/login_token", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			// 判断结果
			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			//登录成功才需要判断
			if resp.Code == http.StatusOK {
				accessToken := resp.Header().Get("x-access-token")
				refreshToken := resp.Header().Get("x-refresh-token")
				acessT, err := userHandle.Jwt.Decrypt(accessToken, web.AccessSecret)
				if err != nil {
					panic(err)
				}
				accessTokenClaim := acessT.(*web.TokenClaims)
				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
					return
				}

				refreshT, err := userHandle.Jwt.Decrypt(refreshToken, web.RefreshSecret)
				if err != nil {
					panic(err)
				}
				refreshTokenClaim := refreshT.(*web.TokenClaims)
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
