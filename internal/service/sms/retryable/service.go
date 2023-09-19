package retryable

import (
	"context"
	"errors"
	"sync/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

// Service 存在并发问题
type Service struct {
	svc sms.Service
	// 重试
	retryMax int
}

func NewService(svc sms.Service, retryMax int) sms.Service {
	return &Service{
		svc:      svc,
		retryMax: retryMax,
	}
}

func (s *Service) Send(ctx context.Context, tpl string, numbers []string, args []sms.ArgVal) error {
	err := s.svc.Send(ctx, tpl, numbers, args)
	cnt := 1
	for err != nil && cnt < s.retryMax {
		err = s.svc.Send(ctx, tpl, numbers, args)
		if err == nil {
			return nil
		}
		cnt++
	}
	return errors.New("重试都失败了")
}

type RetryService struct {
	svcs   []sms.Service
	state  atomic.Uint32
	length int
}

func (r *RetryService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	////TODO implement me

	//第一时间检查是不是有可用服务
	if r.length <= 0 {
		return errors.New("不存在可用的服务")
	}

	return nil
}

// 设计并实现了一个高可用的短信平台
// 1. 提高可用性：重试机制、客户端限流、failover（轮询，实时检测）
// 	1.1 实时检测：
// 	1.1.1 基于超时的实时检测（连续超时）
// 	1.1.2 基于响应时间的实时检测（比如说，平均响应时间上升 20%）
//  1.1.3 基于长尾请求的实时检测（比如说，响应时间超过 1s 的请求占比超过了 10%）
//  1.1.4 错误率
// 2. 提高安全性：
// 	2.1 完整的资源申请与审批流程
//  2.2 鉴权：
// 	2.2.1 静态 token
//  2.2.2 动态 token
// 3. 提高可观测性：日志、metrics, tracing，丰富完善的排查手段
// 4. 提高性能，高性能：

// 我没说怎么实现高并发
