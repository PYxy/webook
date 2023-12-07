package client

import (
	"context"

	"google.golang.org/grpc"

	intrv1 "gitee.com/geekbang/basic-go/webook/api/proto/gen/intr/v1"
	domain2 "gitee.com/geekbang/basic-go/webook/interactive/domain"
	"gitee.com/geekbang/basic-go/webook/interactive/service"
)

// InteractiveServiceAdapter 将一个本地实现伪装成一个 gRPC 客户端
type InteractiveServiceAdapter struct {
	svc service.InteractiveService
}

func NewInteractiveServiceAdapter(svc service.InteractiveService) *InteractiveServiceAdapter {
	return &InteractiveServiceAdapter{svc: svc}
}

func (i *InteractiveServiceAdapter) IncrReadCnt(ctx context.Context, in *intrv1.IncrReadCntRequest, opts ...grpc.CallOption) (*intrv1.IncrReadCntResponse, error) {
	err := i.svc.IncrReadCnt(ctx, in.GetBiz(), in.GetBizId())
	return &intrv1.IncrReadCntResponse{}, err
}

func (i *InteractiveServiceAdapter) Like(ctx context.Context, in *intrv1.LikeRequest, opts ...grpc.CallOption) (*intrv1.LikeResponse, error) {
	err := i.svc.Like(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	return &intrv1.LikeResponse{}, err
}

func (i *InteractiveServiceAdapter) CancelLike(ctx context.Context, in *intrv1.CancelLikeRequest, opts ...grpc.CallOption) (*intrv1.CancelLikeResponse, error) {
	err := i.svc.CancelLike(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	return &intrv1.CancelLikeResponse{}, err
}

func (i *InteractiveServiceAdapter) Collect(ctx context.Context, in *intrv1.CollectRequest, opts ...grpc.CallOption) (*intrv1.CollectResponse, error) {
	err := i.svc.Collect(ctx, in.GetBiz(), in.GetBizId(), in.GetUid(), in.GetCid())
	return &intrv1.CollectResponse{}, err
}

func (i *InteractiveServiceAdapter) Get(ctx context.Context, in *intrv1.GetRequest, opts ...grpc.CallOption) (*intrv1.GetResponse, error) {
	intr, err := i.svc.Get(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	if err != nil {
		return nil, err
	}
	return &intrv1.GetResponse{
		Intr: i.toDTO(intr),
	}, nil
}

func (i *InteractiveServiceAdapter) GetByIds(ctx context.Context, in *intrv1.GetByIdsRequest, opts ...grpc.CallOption) (*intrv1.GetByIdsResponse, error) {
	res, err := i.svc.GetByIds(ctx, in.GetBiz(), in.GetIds())
	if err != nil {
		return nil, err
	}
	m := make(map[int64]*intrv1.Interactive, len(res))
	for k, v := range res {
		m[k] = i.toDTO(v)
	}
	return &intrv1.GetByIdsResponse{
		Intrs: m,
	}, nil
}

// DTO data transfer object
func (i *InteractiveServiceAdapter) toDTO(intr domain2.Interactive) *intrv1.Interactive {
	return &intrv1.Interactive{
		Biz:        intr.Biz,
		BizId:      intr.BizId,
		CollectCnt: intr.CollectCnt,
		Collected:  intr.Collected,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		ReadCnt:    intr.ReadCnt,
	}
}
