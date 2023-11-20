package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/atomic"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
)

type RankingLocalCache struct {
	topN       *atomic.String
	ddl        *atomic.Time
	expiration time.Duration
}

func NewRankingLocalCache() *RankingLocalCache {
	return &RankingLocalCache{
		topN:       atomic.NewString(""),
		ddl:        atomic.NewTime(time.Now()),
		expiration: time.Minute * 10,
	}
}

func (r *RankingLocalCache) Set(ctx context.Context, arts []domain.Article) error {
	// 也可以按照 id => Article 缓存

	for i := 0; i < len(arts); i++ {
		arts[i].Content = ""
	}

	val, err := json.Marshal(arts)
	if err != nil {
		return err
	}
	r.topN.Store(string(val))
	ddl := time.Now().Add(r.expiration)
	r.ddl.Store(ddl)
	return nil
}

func (r *RankingLocalCache) Get(ctx context.Context) ([]domain.Article, error) {
	ddl := r.ddl.Load()
	arts := r.topN.Load()
	var res []domain.Article
	err := json.Unmarshal([]byte(arts), &res)
	if err != nil {
		return nil, err
	}
	if len(arts) == 0 || ddl.Before(time.Now()) {
		return nil, errors.New("本地缓存未命中")
	}
	return res, nil
}
func (r *RankingLocalCache) ForceGet(ctx context.Context) ([]domain.Article, error) {
	arts := r.topN.Load()
	var res []domain.Article
	err := json.Unmarshal([]byte(arts), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type item struct {
	arts []domain.Article
	ddl  time.Time
}
