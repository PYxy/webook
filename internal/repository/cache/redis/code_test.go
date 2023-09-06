package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
	redismocks "gitee.com/geekbang/basic-go/webook/internal/repository/cache/redis/mocks"
)

func TestCodeRedisCache_Set(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctl *gomock.Controller) redis.Cmdable
		wantErr error
		biz     string
		phone   string
		code    string
		cnt     int
	}{
		{
			name: "超时或者断开连接",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(context.DeadlineExceeded)

				rc.EXPECT().Eval(gomock.Any(), luaSetCode, []string{"code:login:13719088200"}, "123456", 600, 90, 3).Return(res)
				return rc
			},
			wantErr: context.DeadlineExceeded,
			biz:     "login",
			phone:   "13719088200",
			code:    "123456",
			cnt:     3,
		},
		{
			name: "保存成功",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(0))
				rc.EXPECT().Eval(gomock.Any(), luaSetCode, []string{"code:login:13719088200"}, "123456", 600, 90, 3).Return(res)
				return rc
			},
			wantErr: nil,
			biz:     "login",
			phone:   "13719088200",
			code:    "123456",
			cnt:     3,
		},
		{
			name: "请求频繁",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(-1))
				res.SetErr(cache.ErrFrequentlyForSend)
				rc.EXPECT().Eval(gomock.Any(), luaSetCode, []string{"code:login:13719088200"}, "123456", 600, 90, 3).Return(res)
				return rc
			},
			wantErr: cache.ErrFrequentlyForSend,
			biz:     "login",
			phone:   "13719088200",
			code:    "123456",
			cnt:     3,
		},
		{
			name: "key过期时间异常",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(-2))
				res.SetErr(cache.ErrUnknownForCode)
				rc.EXPECT().Eval(gomock.Any(), luaSetCode, []string{"code:login:13719088200"}, "123456", 600, 90, 3).Return(res)
				return rc
			},
			wantErr: cache.ErrUnknownForCode,
			biz:     "login",
			phone:   "13719088200",
			code:    "123456",
			cnt:     3,
		},
		{
			name: "系统异常",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(-2))
				res.SetErr(errors.New("系统异常"))
				rc.EXPECT().Eval(gomock.Any(), luaSetCode, []string{"code:login:13719088200"}, "123456", 600, 90, 3).Return(res)
				return rc
			},
			wantErr: errors.New("系统异常"),
			biz:     "login",
			phone:   "13719088200",
			code:    "123456",
			cnt:     3,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			codeCache := NewRedisSmsCache(tc.mock(ctl))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if i == 0 {
				time.Sleep(time.Second)
			}

			err := codeCache.Set(ctx, tc.biz, tc.phone, tc.code, tc.cnt)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestCodeRedisCache_Verify(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctl *gomock.Controller) redis.Cmdable
		wantErr  error
		biz      string
		phone    string
		code     string
		wantflag bool
	}{
		{
			name: "超时或者断开连接",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(context.DeadlineExceeded)

				rc.EXPECT().Eval(gomock.Any(), luaGetCode, []string{"code:login:13719088200"}, "123456").Return(res)
				return rc
			},
			wantErr:  context.DeadlineExceeded,
			biz:      "login",
			phone:    "13719088200",
			code:     "123456",
			wantflag: false,
		},
		{
			name: "验证通过",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(nil)
				res.SetVal(int64(0))
				rc.EXPECT().Eval(gomock.Any(), luaGetCode, []string{"code:login:13719088200"}, "123456").Return(res)
				return rc
			},
			wantErr:  nil,
			biz:      "login",
			phone:    "13719088200",
			code:     "123456",
			wantflag: true,
		},
		{
			name: "正常来说，如果频繁出现这个错误，你就要告警，因为有人搞你",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(cache.ErrCodeVerifyTooManyTimes)
				res.SetVal(int64(-1))
				rc.EXPECT().Eval(gomock.Any(), luaGetCode, []string{"code:login:13719088200"}, "123456").Return(res)
				return rc
			},
			wantErr:  cache.ErrCodeVerifyTooManyTimes,
			biz:      "login",
			phone:    "13719088200",
			code:     "123456",
			wantflag: false,
		},
		{
			name: "密码输错了(3次之内)",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(nil)
				res.SetVal(int64(-2))
				rc.EXPECT().Eval(gomock.Any(), luaGetCode, []string{"code:login:13719088200"}, "123456").Return(res)
				return rc
			},
			wantErr:  nil,
			biz:      "login",
			phone:    "13719088200",
			code:     "123456",
			wantflag: false,
		},
		{
			name: "获取验证码之后  修改电话号码",
			mock: func(ctl *gomock.Controller) redis.Cmdable {
				rc := redismocks.NewMockCmdable(ctl)
				//设置响应体
				res := redis.NewCmd(context.Background())
				res.SetErr(cache.ErrAttack)
				res.SetVal(int64(-3))
				rc.EXPECT().Eval(gomock.Any(), luaGetCode, []string{"code:login:13719088200"}, "123456").Return(res)
				return rc
			},
			wantErr:  cache.ErrAttack,
			biz:      "login",
			phone:    "13719088200",
			code:     "123456",
			wantflag: false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			codeCache := NewRedisSmsCache(tc.mock(ctl))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if i == 0 {
				time.Sleep(time.Second)
			}

			flag, err := codeCache.Verify(ctx, tc.biz, tc.phone, tc.code)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantflag, flag)

		})
	}
}
