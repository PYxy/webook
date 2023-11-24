package repository

import (
	"context"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
)

type JobRepository interface {
	Preempt(ctx context.Context) (domain.Job, error)
	Release(ctx context.Context, id int64, version int) error
	UpdateUtime(ctx context.Context, id int64, version int) error
	UpdateNextTime(ctx context.Context, id int64, next time.Time, version int) error
	Stop(ctx context.Context, id int64, version int) error
}

type PreemptCronJobRepository struct {
	dao dao.JobDAO
}

func (p *PreemptCronJobRepository) UpdateUtime(ctx context.Context, id int64, version int) error {
	return p.dao.UpdateUtime(ctx, id, version)
}

func (p *PreemptCronJobRepository) UpdateNextTime(ctx context.Context, id int64, next time.Time, version int) error {
	return p.dao.UpdateNextTime(ctx, id, next, version)
}

func (p *PreemptCronJobRepository) Stop(ctx context.Context, id int64, version int) error {
	return p.dao.Stop(ctx, id, version)
}

func (p *PreemptCronJobRepository) Release(ctx context.Context, id int64, version int) error {
	return p.dao.Release(ctx, id, version)
}

func (p *PreemptCronJobRepository) Preempt(ctx context.Context) (domain.Job, error) {
	j, err := p.dao.Preempt(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	return domain.Job{
		Cfg:      j.Cfg,
		Id:       j.Id,
		Name:     j.Name,
		Version:  j.Version,
		Executor: j.Executor,
	}, nil
}
