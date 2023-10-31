package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type InteractiveService interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// Like 点赞
	Like(ctx context.Context, biz string, bizId int64, uid int64) error
	// CancelLike 取消点赞
	CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error
	// Collect 收藏
	Collect(ctx context.Context, biz string, bizId, cid, uid int64) error
	Get(ctx context.Context, biz string, bizId, uid int64) (domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
	l    logger.LoggerV1
}

func (i *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncrReadCnt(ctx, biz, bizId)
}

func (i *interactiveService) Get(
	ctx context.Context, biz string, bizId, uid int64) (domain.Interactive, error) {
	// 按照 repository 的语义(完成 domain.Interactive 的完整构造)，你这里拿到的就应该是包含全部字段的
	var (
		eg        errgroup.Group
		intr      domain.Interactive
		liked     bool
		collected bool
	)
	eg.Go(func() error {
		var err error
		intr, err = i.repo.Get(ctx, biz, bizId)
		return err
	})
	//判断当前用户是否有点赞
	eg.Go(func() error {
		var err error
		liked, err = i.repo.Liked(ctx, biz, bizId, uid)
		return err
	})
	//判断当前用户是否有收藏
	eg.Go(func() error {
		var err error
		collected, err = i.repo.Collected(ctx, biz, bizId, uid)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return domain.Interactive{}, err
	}
	intr.Liked = liked
	intr.Collected = collected
	return intr, nil

}

func (i *interactiveService) Like(ctx context.Context, biz string, bizId int64, uid int64) error {
	// 点赞
	return i.repo.IncrLike(ctx, biz, bizId, uid)
}

func (i *interactiveService) CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error {
	return i.repo.DecrLike(ctx, biz, bizId, uid)
}

// Collect 收藏
func (i *interactiveService) Collect(ctx context.Context,
	biz string, bizId, cid, uid int64) error {
	// service 还叫做收藏
	// repository
	return i.repo.AddCollectionItem(ctx, biz, bizId, cid, uid)
}

func NewInteractiveService(repo repository.InteractiveRepository,
	l logger.LoggerV1) InteractiveService {
	return &interactiveService{
		repo: repo,
		l:    l,
	}
}
