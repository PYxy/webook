package service

import (
	"context"
	"errors"
	"fmt"
	accountv1 "gitee.com/geekbang/basic-go/webook/api/proto/gen/account/v1"
	pmtv1 "gitee.com/geekbang/basic-go/webook/api/proto/gen/payment/v1"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/reward/domain"
	"gitee.com/geekbang/basic-go/webook/reward/repository"
	"strconv"
	"strings"
)

type WechatNativeRewardService struct {
	client pmtv1.WechatPaymentServiceClient
	repo   repository.RewardRepository
	l      logger.LoggerV1
	acli   accountv1.AccountServiceClient
}

func (s *WechatNativeRewardService) UpdateReward(ctx context.Context,
	bizTradeNO string, status domain.RewardStatus) error {
	rid := s.toRid(bizTradeNO)
	err := s.repo.UpdateStatus(ctx, rid, status)
	if err != nil {
		return err
	}
	// 完成了支付，准备入账
	if status == domain.RewardStatusPayed {
		r, err := s.repo.GetReward(ctx, rid)
		if err != nil {
			return err
		}
		// webook 抽成
		weAmt := int64(float64(r.Amt) * 0.1)
		_, err = s.acli.Credit(ctx, &accountv1.CreditRequest{
			Biz:   "reward",
			BizId: rid,
			Items: []*accountv1.CreditItem{
				{
					AccountType: accountv1.AccountType_AccountTypeReward,
					// 虽然可能为 0，但是也要记录出来
					Amt:      weAmt,
					Currency: "CNY",
				},
				{
					Account:     r.Uid,
					Uid:         r.Uid,
					AccountType: accountv1.AccountType_AccountTypeReward,
					Amt:         r.Amt - weAmt,
					Currency:    "CNY",
				},
			},
		})
		if err != nil {
			s.l.Error("入账失败了，快来修数据啊！！！",
				logger.String("biz_trade_no", bizTradeNO),
				logger.Error(err))
			// 做好监控和告警，这里
			return err
		}
	}
	return nil
}

func (s *WechatNativeRewardService) GetReward(ctx context.Context, rid, uid int64) (domain.Reward, error) {
	// 快路径
	r, err := s.repo.GetReward(ctx, rid)
	if err != nil {
		return domain.Reward{}, err
	}
	if r.Uid != uid {
		// 说明是非法查询
		return domain.Reward{}, errors.New("查询的打赏记录和打赏人对不上")
	}
	// 已经是完结状态
	if r.Completed() {
		return r, nil
	}
	// 这个时候，考虑到支付到查询结果，我们搞一个慢路径
	resp, err := s.client.GetPayment(ctx, &pmtv1.GetPaymentRequest{
		BizTradeNo: s.bizTradeNO(r.Id),
	})
	if err != nil {
		// 这边我们直接返回从数据库查询的数据
		s.l.Error("慢路径查询支付结果失败",
			logger.Int64("rid", r.Id), logger.Error(err))
		return r, nil
	}
	// 更新状态
	switch resp.Status {
	case pmtv1.PaymentStatus_PaymentStatusFailed:
		r.Status = domain.RewardStatusFailed
	case pmtv1.PaymentStatus_PaymentStatusInit:
		r.Status = domain.RewardStatusInit
	case pmtv1.PaymentStatus_PaymentStatusSuccess:
		r.Status = domain.RewardStatusPayed
	case pmtv1.PaymentStatus_PaymentStatusRefund:
		// 理论上来说不可能出现这个，直接设置为失败
		r.Status = domain.RewardStatusFailed
	}
	err = s.repo.UpdateStatus(ctx, rid, r.Status)
	if err != nil {
		s.l.Error("更新本地打赏状态失败",
			logger.Int64("rid", r.Id), logger.Error(err))
		return r, nil
	}
	return r, nil
}

func (s *WechatNativeRewardService) PreReward(ctx context.Context, r domain.Reward) (domain.CodeURL, error) {
	// 先查询缓存，确认是否已经创建过了打赏的预支付订单
	codeUrl, err := s.repo.GetCachedCodeURL(ctx, r)
	if err == nil {
		return codeUrl, nil
	}
	r.Status = domain.RewardStatusInit
	rid, err := s.repo.CreateReward(ctx, r)
	if err != nil {
		return domain.CodeURL{}, err
	}
	resp, err := s.client.NativePrePay(ctx, &pmtv1.PrePayRequest{
		Amt: &pmtv1.Amount{
			Total:    r.Amt,
			Currency: "CNY",
		},
		// 想办法拼接出来一个 biz_trade_id
		BizTradeNo:  fmt.Sprintf("reward-%d", rid),
		Description: fmt.Sprintf("打赏-%s", r.Target.BizName),
	})
	if err != nil {
		return domain.CodeURL{}, err
	}
	cu := domain.CodeURL{
		Rid: rid,
		URL: resp.CodeUrl,
	}
	err1 := s.repo.CachedCodeURL(ctx, cu, r)
	if err1 != nil {
		s.l.Error("缓存二维码失败", logger.Error(err1))
	}
	return cu, err
}

func (s *WechatNativeRewardService) bizTradeNO(rid int64) string {
	return fmt.Sprintf("reward-%d", rid)
}

func (s *WechatNativeRewardService) toRid(tradeNO string) int64 {
	ridStr := strings.Split(tradeNO, "-")
	val, _ := strconv.ParseInt(ridStr[1], 10, 64)
	return val
}

func NewWechatNativeRewardService(
	client pmtv1.WechatPaymentServiceClient,
	repo repository.RewardRepository,
	l logger.LoggerV1,
	acli accountv1.AccountServiceClient,
) RewardService {
	return &WechatNativeRewardService{client: client, repo: repo, l: l, acli: acli}
}
