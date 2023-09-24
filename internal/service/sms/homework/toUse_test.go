package homework

import (
	"context"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"github.com/ecodeclub/ekit/retry"
	"gorm.io/gorm"
	"testing"
	"time"
)

// TestAll 未测试 纯模板
func TestAll(t *testing.T) {
	var a sms.Service
	var l LocalWindow
	monitor := NewMonitorService(a, l, 10, (time.Millisecond * 400).Milliseconds(), time.Second.Microseconds())
	var db *gorm.DB
	smsDao := dao.NewSmsDao(db)
	repo := repository.NewSMSRepo(smsDao)
	//v1
	//availability := NewAvailability([]MonitorServiceInterface{monitor}, time.Minute*2, repo, func() Strategy {
	//	strategy, _ := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
	//	return strategy
	//})
	//
	//availability.Async()
	//availability.AsyncToCheck()
	//v2
	smsService := NewAvailabilityV1([]MonitorServiceInterface{monitor}, time.Minute*2, repo, func() Strategy {
		strategy, _ := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
		return strategy
	}, func() Random {
		return &MyRandom{}
	})
	//V2 可以变成service
	var (
		biz          string
		phoneNumbers []string
		args         []sms.ArgVal
	)

	_ = smsService.Send(context.Background(), biz, phoneNumbers, args)

	//select {}
}
