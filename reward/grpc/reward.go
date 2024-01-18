package grpc

import (
	"context"
	"gitee.com/geekbang/basic-go/webook/api/proto/gen/reward/v1"
	"gitee.com/geekbang/basic-go/webook/reward/domain"
	"gitee.com/geekbang/basic-go/webook/reward/service"
	"google.golang.org/grpc"
)

type RewardServiceServer struct {
	rewardv1.UnimplementedRewardServiceServer
	svc service.RewardService
}

func NewRewardServiceServer(svc service.RewardService) *RewardServiceServer {
	return &RewardServiceServer{svc: svc}
}

func (r *RewardServiceServer) Register(server *grpc.Server) {
	rewardv1.RegisterRewardServiceServer(server, r)
}

func (r *RewardServiceServer) PreReward(ctx context.Context, request *rewardv1.PreRewardRequest) (*rewardv1.PreRewardResponse, error) {
	codeURL, err := r.svc.PreReward(ctx, domain.Reward{
		Uid: request.Uid,
		Target: domain.Target{
			Biz:     request.Biz,
			BizId:   request.BizId,
			BizName: request.BizName,
			Uid:     request.Uid,
		},
		Amt: request.Amt,
	})
	return &rewardv1.PreRewardResponse{
		CodeUrl: codeURL.URL,
		Rid:     codeURL.Rid,
	}, err
}

func (r *RewardServiceServer) GetReward(ctx context.Context,
	req *rewardv1.GetRewardRequest) (*rewardv1.GetRewardResponse, error) {
	rw, err := r.svc.GetReward(ctx, req.GetRid(), req.GetUid())
	if err != nil {
		return nil, err
	}
	return &rewardv1.GetRewardResponse{
		// 两个的取值是一样的，所以可以直接转
		Status: rewardv1.RewardStatus(rw.Status),
	}, nil
}
