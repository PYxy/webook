package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/ecodeclub/ekit/queue"
	"github.com/ecodeclub/ekit/slice"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
)

var ErrOutOfCapacity = errors.New("ekit: 超出最大容量限制")

type RankingService interface {
	TopN(ctx context.Context) error
	//方便测试
	//TopN(ctx context.Context, n int64) ([]domain.Article, error)
}

// BatchRankingService 排行榜 定时计算
type BatchRankingService struct {
	artSvc  ArticleService
	intrSvc InteractiveService
	repo    repository.RankingRepository
	// 多少条数据为一批
	batchSize int
	// 表示需要排名TOP n 的个数(前N名)
	n int

	scoreFunc func(time.Time, int64) float64
}

// 准备分批
func (svc *BatchRankingService) TopN(ctx context.Context) error {
	arts, err := svc.topN(ctx)
	if err != nil {
		return err
	}
	// 在这里，存起来
	return svc.repo.ReplaceTopN(ctx, arts)
}

func (svc *BatchRankingService) topN(ctx context.Context) ([]domain.Article, error) {
	// 我只取七天内的数据
	now := time.Now()
	// 先拿一批数据
	offset := 0
	type Score struct {
		art   domain.Article
		score float64
	}
	// 这里可以用非并发安全
	topN := queue.NewConcurrentPriorityQueue[Score](svc.n,
		func(src Score, dst Score) int {
			if src.score > dst.score {
				return 1
			} else if src.score == dst.score {
				return 0
			} else {
				return -1
			}
		})

	for {
		// 这里拿了一批
		//获取所有的文章列表
		arts, err := svc.artSvc.ListPub(ctx, now, offset, svc.batchSize)
		if err != nil {
			return nil, err
		}
		ids := slice.Map[domain.Article, int64](arts,
			func(idx int, src domain.Article) int64 {
				return src.Id
			})
		// 要去找到对应的点赞数据
		intrs, err := svc.intrSvc.GetByIds(ctx, "article", ids)
		if err != nil {
			return nil, err
		}
		// 合并计算 score
		// 排序
		for _, art := range arts {
			//根据id 获取到对应的点赞对象
			intr := intrs[art.Id]
			//if !ok {
			//	// 你都没有，肯定不可能是热榜
			//	continue
			//}
			score := svc.scoreFunc(art.Utime, intr.LikeCnt)
			// 我要考虑，我这个 score 在不在前一百名
			// 拿到热度最低的
			err = topN.Enqueue(Score{
				art:   art,
				score: score,
			})
			// 这种写法，要求 topN 已经满了
			if err == ErrOutOfCapacity {
				//这里不应该是直接dequeue 应该写一个方法 获取最小的值 但是不出队 减少消耗
				val, _ := topN.Dequeue()
				if val.score < score {
					err = topN.Enqueue(Score{
						art:   art,
						score: score,
					})
				} else {
					_ = topN.Enqueue(val)
				}
			}
		}

		// 一批已经处理完了，问题来了，我要不要进入下一批？我怎么知道还有没有？
		if len(arts) < svc.batchSize ||
			//只获取7天之内更新的数据
			now.Sub(arts[len(arts)-1].Utime).Hours() < 7*24 {
			// 我这一批都没取够，我当然可以肯定没有下一批了
			break
		}
		// 这边要更新 offset 方便下次从上一次结束的敌法开始
		offset = offset + len(arts)
	}
	// 最后得出结果
	res := make([]domain.Article, svc.n)
	for i := svc.n - 1; i >= 0; i-- {
		val, err := topN.Dequeue()
		if err != nil {
			// 说明取完了，不够 n
			break
		}
		res[i] = val.art
	}
	return res, nil
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
