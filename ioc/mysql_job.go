package ioc

import (
	"context"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/job"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

// 封装
func InitScheduler(l logger.LoggerV1,
	local *job.LocalFuncExecutor,
	svc service.JobService, name string) *job.Scheduler {
	res := job.NewScheduler(svc, l, name)
	res.RegisterExecutor(local)
	return res
}

// InitLocalFuncExecutor 定时定时任务处理器
func InitLocalFuncExecutor(svc service.RankingService) *job.LocalFuncExecutor {
	res := job.NewLocalFuncExecutor()
	// 要在数据库里面插入一条记录。
	// ranking job 的记录，通过管理任务接口来插入
	res.RegisterFunc("ranking", func(ctx context.Context, j domain.Job) error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		return svc.TopN(ctx)
	})
	return res
}
