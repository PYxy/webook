package auth

import (
	"context"
	"testing"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"gitee.com/geekbang/basic-go/webook/internal/service/sms/aliyun"
	"gitee.com/geekbang/basic-go/webook/internal/service/sms/ratelimit"
)

// TestAuthSMSService_Send 1.使用细节 先Auth SMS 鉴权一下 再限流 SMS (因为都实现了 sms.Service 接口)
func TestAuthSMSService_Send(t *testing.T) {
	//发送短信的核心
	smsService := aliyun.NewAliyunService("", "", "", "", "")
	//限流器装饰器
	limit := ratelimit.NewRedisSlidingWindowLimiter(nil, time.Second, 10)
	limitService := ratelimit.NewRateLimitSMSService(smsService, limit)
	//鉴权装饰器
	authService := NewAuthSMSService(limitService, "密钥")

	authService.Send(context.Background(), "加密密钥", []string{}, []sms.ArgVal{})
}
