package service

import (
	"context"
	"math"
	"time"
)

type RankingService interface {
	TopN(ctx context.Context) error
	//方便测试
	//TopN(ctx context.Context, n int64) ([]domain.Article, error)
}

// BatchRankingService 排行榜 定时计算
type BatchRankingService struct {
	artSvc  ArticleService
	intrSvc InteractiveService
	// 多少条数据为一批
	batchSize int
	// topN
	n int

	scoreFunc func(time.Time, int64) float64
}

func (b *BatchRankingService) TopN(ctx context.Context) error {
	return nil
}

func NewBatchRankingService(artSvc ArticleService, interSvc InteractiveService, batchSize, topN int) *BatchRankingService {
	return &BatchRankingService{
		artSvc:    artSvc,
		intrSvc:   interSvc,
		batchSize: batchSize,
		n:         topN,
		scoreFunc: func(t time.Time, likeCnt int64) float64 {
			return float64(likeCnt-1) / math.Pow(float64(likeCnt+2), 1.5)
		},
	}
}
