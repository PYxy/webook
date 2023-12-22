package service

import (
	"context"
	"sync/atomic"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type JobService interface {
	// Preempt 抢占
	Preempt(ctx context.Context) (domain.Job, error)
	ResetNextTime(ctx context.Context, j domain.Job) error
	NeedToReset() bool
	// 我返回一个释放的方法，然后调用者取调
	// PreemptV1(ctx context.Context) (domain.Job, func() error,  error)
	// Release
	//Release(ctx context.Context, id int64) error
}

type cronJobService struct {
	repo            repository.JobRepository
	refreshInterval time.Duration
	l               logger.LoggerV1
	needToUpdate    *atomic.Bool //没必要
}

func (p *cronJobService) NeedToReset() bool {
	return p.needToUpdate.Load()
}

func (p *cronJobService) Preempt(ctx context.Context) (domain.Job, error) {
	j, err := p.repo.Preempt(ctx)
	if err != nil {
		return j, err
	}

	// 你的续约呢?
	//ch := make(chan struct{})
	//go func() {
	//	ticker := time.NewTicker(p.refreshInterval)
	//	for {
	//		select {
	//		case <-ticker.C:
	//			// 在这里续约
	//			p.refresh(j.Id)
	//		case <-ch:
	//			// 结束
	//			return
	//		}
	//	}
	//}()

	ticker := time.NewTicker(p.refreshInterval)

	go func() {
		for range ticker.C {
			//这里可以给重试
			err2 := p.refresh(j.Id, j.Version)
			if err2 != nil {
				p.needToUpdate.Store(false)
				break
			}
		}
	}()

	// 你抢占之后，你一直抢占着吗？
	// 你要考虑一个释放的问题
	j.CancelFunc = func() error {
		//close(ch)
		// 自己在这里释放掉
		ticker.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return p.repo.Release(ctx, j.Id, j.Version)
	}
	return j, err
}

func (p *cronJobService) ResetNextTime(ctx context.Context, j domain.Job) error {
	next := j.NextTime()
	if next.IsZero() {
		// 没有下一次
		return p.repo.Stop(ctx, j.Id, j.Version)
	}
	return p.repo.UpdateNextTime(ctx, j.Id, next, j.Version)
}

func (p *cronJobService) refresh(id int64, version int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 续约怎么个续法？
	// 更新一下更新时间就可以
	// 比如说我们的续约失败逻辑就是：处于 running 状态，但是更新时间在三分钟以前
	err := p.repo.UpdateUtime(ctx, id, version)
	if err != nil {
		// 可以考虑立刻重试
		p.l.Error("续约失败",
			logger.Error(err),
			logger.Int64("jid", id))
	}

	return err
}
