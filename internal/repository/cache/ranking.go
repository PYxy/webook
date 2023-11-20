package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
)

type RankingCache interface {
	Set(ctx context.Context, arts []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RankingRedisCache struct {
	client redis.Cmdable
	key    string
}

func (r *RankingRedisCache) Set(ctx context.Context, arts []domain.Article) error {
	//根据业务缓存思路 topN 肯定是多人点开的
	// id=> article
	for i := 0; i < len(arts); i++ {
		arts[i].Content = ""
	}

	val, err := json.Marshal(arts)
	if err != nil {
		return err
	}
	// 这个过期时间要稍微长一点，最好是超过计算热榜的时间（包含重试在内的时间）
	// 你甚至可以直接永不过期
	return r.client.Set(ctx, r.key, val, time.Minute*10).Err()
}

func (r *RankingRedisCache) Get(ctx context.Context) ([]domain.Article, error) {
	data, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		return nil, err
	}

	var res []domain.Article
	err = json.Unmarshal(data, &res)
	return res, err
}
