package ratelimit

import (
	"context"
	"fmt"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"gitee.com/geekbang/basic-go/webook/pkg/ratelimit"
)

var errLimited = fmt.Errorf("触发了限流")

type RateLimitSMSService struct {
	svc     sms.Service
	limiter ratelimit.Limiter
}

func NewRateLimitSMSService(svc sms.Service, limiter ratelimit.Limiter) *RateLimitSMSService {
	return &RateLimitSMSService{
		svc:     svc,
		limiter: limiter,
	}
}

// Send 在阿里云的接口中
// TPL  是业务类型 不是JWT
func (s *RateLimitSMSService) Send(ctx context.Context, tpl string, phoneNumbers []string, args []sms.ArgVal) error {
	limit, err := s.limiter.Limit(ctx, tpl)
	if err != nil {
		return err
	}
	if limit {
		return errLimited
	}
	return s.svc.Send(ctx, tpl, phoneNumbers, args)
}
