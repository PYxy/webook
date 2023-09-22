package homework

import (
	"context"
	"strings"
	"time"

	"go.uber.org/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"

	"github.com/demdxx/gocast/v2"
)

//
//import (
//	"context"
//	"errors"
//	"math/rand"
//	"sync/atomic"
//	"time"
//
//	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
//)
//
//func init() {
//	rand.Seed(time.Now().UnixNano())
//}
//
//type inner struct {
//	percentage int
//	svc        sms.Service
//}
//
//func (i *inner) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
//	//TODO implement me
//	return i.svc.Send(ctx, biz, phoneNumbers, args)
//}
//
//type RetryService struct {
//	//服务列表
//	svcs []inner
//	//记录服务列表的状态
//	state uint32
//	//总服务个数
//	length uint64
//	// 处理策略分拣者
//	idx      uint64
//	operator OPERATOR
//}
//
//func (r *RetryService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
//	//TODO implement me
//
//	//第一时间检查是不是有可用服务
//	if r.length <= 0 {
//		return errors.New("不存在可用的服务")
//	}
//	if flag, _ := r.operator.Diversion(); flag {
//		//走异步检测坏的实例
//	}
//
//	//正常流程
//	serviceIndex := atomic.LoadUint32(&r.state)
//	//获取正常的实例
//
//	idx := atomic.AddUint64(&r.idx, 1)
//	for i := idx; i < r.length+idx; i++ {
//		if serviceIndex&(1<<i) == 0 {
//			//不健康的服务
//			continue
//		} else {
//			//健康的服务
//			//检查是不是刚恢复的设备
//			innerSvc := r.svcs[i]
//			if rand.Intn(11) <= innerSvc.percentage {
//				err := innerSvc.Send(ctx, biz, phoneNumbers, args)
//			}
//		}
//	}
//
//	return nil
//}
//
//// 设计并实现了一个高可用的短信平台
//// 1. 提高可用性：重试机制、客户端限流、failover（轮询，实时检测）
//// 	1.1 实时检测：
//// 	1.1.1 基于超时的实时检测（连续超时）
//// 	1.1.2 基于响应时间的实时检测（比如说，平均响应时间上升 20%）
////  1.1.3 基于长尾请求的实时检测（比如说，响应时间超过 1s 的请求占比超过了 10%）
////  1.1.4 错误率
//// 2. 提高安全性：
//// 	2.1 完整的资源申请与审批流程
////  2.2 鉴权：
//// 	2.2.1 静态 token
////  2.2.2 动态 token
//// 3. 提高可观测性：日志、metrics, tracing，丰富完善的排查手段
//// 4. 提高性能，高性能：
//
//// 我没说怎么实现高并发

type Efect struct {
	//服务列表
	svcs []NewService
	//记录服务列表的状态
	state uint32
	//总服务个数
	length uint64
	// 处理策略分拣者
	idx uint64

	//存在有坏的设备的
	needToCheck atomic.String
	//检测时间间隔
	interval time.Duration
}

func (e *Efect) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO implement me
	return nil
}

//func (e *Efect) Async() {
//	go func() {
//		for {
//			result := e.needToCheck.Load()
//			if result != "" {
//				// 切割  "2,3"  分别代表索引的下标
//				for _, val := range CutString(result) {
//					if e.svcs[val].Healthy() {
//						for {
//							//这里可以一直抢 因为这个位置是检查坏的  正常的流程是不会 把坏的位置给占的(坏实例变好)
//							tmpRes := at.LoadUint32(&e.state)
//							if at.CompareAndSwapUint32(&e.state, tmpRes, tmpRes|1<<val) {
//								break
//							}
//						}
//
//					}
//				}
//			}
//			//无论什么情况都休眠
//			time.Sleep(e.interval)
//		}
//	}()
//}

func CutString(indexStr string) []int {
	return gocast.ToIntSlice(strings.Split(indexStr, ","))
}
