package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	svcmocks "gitee.com/geekbang/basic-go/webook/internal/service/mocks"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

func TestArticleHandler_Publish(t *testing.T) {
	const url = "/articles/publish"
	testCases := []struct {
		name       string
		mock       func(ctrl *gomock.Controller) service.ArticleService
		reqBody    string
		reqBuilder func(t *testing.T, reqBody string) *http.Request

		resCode int64
		wantRes Result
	}{
		{
			name: "正常请求",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				svc := svcmocks.NewMockArticleService(ctrl)
				svc.EXPECT().Publish(gomock.Any(), domain.Article{
					Id:      0,
					Title:   "我的标题",
					Content: "我的内容",
					Author:  domain.Author{Id: 123},
				}).Return(int64(1), nil)

				return svc
			},
			reqBody: `
{
	"title":"我的标题",
	"content": "我的内容"
}`,
			reqBuilder: func(t *testing.T, reqBody string) *http.Request {
				req, err := http.NewRequest(http.MethodPost, url,
					bytes.NewReader([]byte(reqBody)),
				)
				require.NoError(t, err)

				return req
			},

			resCode: http.StatusOK,
			wantRes: Result{
				Msg: "OK",
				//// 在 json 反序列化的时候，因为 Data 是 any，所以默认是 float64
				Data: float64(1),
			},
		},
		{
			name: "更新已有的帖子成功",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				svc := svcmocks.NewMockArticleService(ctrl)
				svc.EXPECT().Publish(gomock.Any(), domain.Article{
					Id:      157,
					Title:   "我的标题",
					Content: "我的内容",
					Author:  domain.Author{Id: 123},
				}).Return(int64(157), nil)

				return svc
			},
			reqBody: `
{
	"Id": 157,
	"title":"我的标题",
	"content": "我的内容"
}`,
			reqBuilder: func(t *testing.T, reqBody string) *http.Request {
				req, err := http.NewRequest(http.MethodPost, url,
					bytes.NewReader([]byte(reqBody)),
				)
				require.NoError(t, err)

				return req
			},

			resCode: http.StatusOK,
			wantRes: Result{
				Msg:  "OK",
				Data: float64(157),
			},
		},
		{
			name: "参数绑定异常",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				svc := svcmocks.NewMockArticleService(ctrl)

				return svc
			},
			reqBody: `
{
	"title":"我的标题",
	"content": "我的内容"
`,
			reqBuilder: func(t *testing.T, reqBody string) *http.Request {
				req, err := http.NewRequest(http.MethodPost, url,
					bytes.NewReader([]byte(reqBody)),
				)
				require.NoError(t, err)

				return req
			},
			resCode: http.StatusBadRequest,
		},
		{
			name: "发表帖子失败",
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				svc := svcmocks.NewMockArticleService(ctrl)
				svc.EXPECT().Publish(gomock.Any(), domain.Article{
					Id:      0,
					Title:   "我的标题",
					Content: "我的内容",
					Author:  domain.Author{Id: 123},
				}).Return(int64(0), errors.New("发表帖子失败"))

				return svc
			},
			reqBody: `
{
	"title":"我的标题",
	"content": "我的内容"
}`,
			reqBuilder: func(t *testing.T, reqBody string) *http.Request {
				req, err := http.NewRequest(http.MethodPost, url,
					bytes.NewReader([]byte(reqBody)),
				)
				require.NoError(t, err)

				return req
			},

			resCode: http.StatusOK,
			wantRes: Result{
				Code: 5,
				Msg:  "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc := tc.mock(ctrl)
			handle := NewArticleHandler(svc, logger.NewNoOpLogger())
			// 注册路由
			server := gin.Default()
			// 设置登录态
			server.Use(func(ctx *gin.Context) {
				ctx.Set("claims", &LUserClaims{
					Uid: 123,
				})
			})
			handle.RegisterRoutes(server)

			req := tc.reqBuilder(t, tc.reqBody)
			req.Header.Set("Content-Type", "application/json")
			// 准备记录响应
			recorder := httptest.NewRecorder()
			// 执行
			server.ServeHTTP(recorder, req)

			// 断言
			assert.Equal(t, tc.resCode, int64(recorder.Code))
			if recorder.Code != http.StatusOK {
				return
			}
			var res Result
			err := json.Unmarshal(recorder.Body.Bytes(), &res)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRes, res)

			//assert.Equal(t, tc.result.Code, CurrentResult.Code)
			//assert.Equal(t, tc.result.Data, CurrentResult.Data)
		})
	}
}
