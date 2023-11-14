package cache

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
)

var (
	//go:embed lua/interative_incr_cnt.lua
	luaIncrCnt string
)
var ErrKeyNotExist = redis.Nil

const (
	fieldReadCnt    = "read_cnt"
	fieldCollectCnt = "collect_cnt"
	fieldLikeCnt    = "like_cnt"
)

//go:generate mockgen -source=./interactive.go -package=cachemocks -destination=mocks/interactive.mock.go InteractiveCache
type InteractiveCache interface {

	// IncrReadCntIfPresent 如果在缓存中有对应的数据，就 +1
	IncrReadCntIfPresent(ctx context.Context,
		biz string, bizId int64) error
	IncrLikeCntIfPresent(ctx context.Context,
		biz string, bizId int64) error
	DecrLikeCntIfPresent(ctx context.Context,
		biz string, bizId int64) error
	IncrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error
	// Get 查询缓存中数据
	Get(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Set(ctx context.Context, biz string, bizId int64, intr domain.Interactive) error
	SetTopN(ctx context.Context, key string, res []domain.TopInteractive, removeKeys []string) error
	GetTopN(ctx context.Context, key string, top int64) ([]domain.TopInteractive, error)
}

// 方案1
// key1 => map[string]int

// 方案2
// key1_read_cnt => 10
// key1_collect_cnt => 11
// key1_like_cnt => 13

type RedisInteractiveCache struct {
	client     redis.Cmdable
	expiration time.Duration
	tracer     trace.Tracer
}

func (r *RedisInteractiveCache) GetTopN(ctx context.Context, key string, top int64) ([]domain.TopInteractive, error) {
	//TODO implement me
	//trace.ContextWithSpan()
	CtxSpen, span := r.tracer.Start(ctx, "redis-cache")
	span.AddEvent("cache-event")
	defer span.End()

	res, err := r.client.ZRevRangeWithScores(CtxSpen, key, 0, top).Result()
	//defer span.End()
	//即使key 不存在 也不会返回err
	if err != nil {
		span.SetAttributes(attribute.String(key, err.Error()))
		return nil, err

	}
	if err == nil && len(res) == 0 {
		span.SetAttributes(attribute.String(key, "key 不存在"))
		return nil, ErrKeyNotExist
	}
	var topInteractive []domain.TopInteractive
	for _, val := range res {
		str := val.Member.(string)
		seps := strings.Split(str, ":")
		//fmt.Printf("id:%v biz_id:%v biz:%v score:%v  \n", seps[0], seps[1], seps[2], val.Score)
		id, biz_id, biz, score := seps[0], seps[1], seps[2], val.Score
		topInteractive = append(topInteractive, domain.TopInteractive{
			Id:      gocast.Int64(id),
			BizId:   gocast.Int64(biz_id),
			Biz:     biz,
			ReadCnt: 0,
			LikeCnt: gocast.Int64(score),
		})
		//fmt.Println(val),
	}
	return topInteractive, nil
}

func (r *RedisInteractiveCache) SetTopN(ctx context.Context, key string, res []domain.TopInteractive, removeKeys []string) error {
	var zset []redis.Z
	for _, topInteractive := range res {
		zset = append(zset,
			redis.Z{
				Score:  float64(topInteractive.LikeCnt),
				Member: fmt.Sprintf("%v:%v:%v", topInteractive.Id, topInteractive.BizId, topInteractive.Biz),
			})
	}
	go func() {
		fmt.Println("需要删除的keys:", removeKeys)
		if len(removeKeys) != 0 {
			//记录异常
			ct, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer func() {
				cancel()
			}()
			fmt.Println("删除多余的key:", r.client.ZRem(ct, key, removeKeys).Err())
		}
	}()
	return r.client.ZAdd(ctx, key, zset...).Err()

}

func (r *RedisInteractiveCache) IncrCollectCntIfPresent(ctx context.Context,
	biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId),
		fieldCollectCnt}, 1).Err()
}

func (r *RedisInteractiveCache) IncrReadCntIfPresent(ctx context.Context,
	biz string, bizId int64) error {
	// 拿到的结果，可能自增成功了，可能不需要自增（key不存在）
	// 你要不要返回一个 error 表达 key 不存在？
	//res, err := r.client.Eval(ctx, luaIncrCnt,
	//	[]string{r.key(biz, bizId)},
	//	// read_cnt +1
	//	"read_cnt", 1).Int()
	//if err != nil {
	//	return err
	//}
	//if res == 0 {
	// 这边一般是缓存过期了
	//	return errors.New("缓存中 key 不存在")
	//}
	return r.client.Eval(ctx, luaIncrCnt,
		[]string{r.key(biz, bizId)},
		// read_cnt +1
		fieldReadCnt, 1).Err()
}

func (r *RedisInteractiveCache) IncrLikeCntIfPresent(ctx context.Context,
	biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt,
		[]string{r.key(biz, bizId)},
		fieldLikeCnt, 1).Err()
}

func (r *RedisInteractiveCache) DecrLikeCntIfPresent(ctx context.Context,
	biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt,
		[]string{r.key(biz, bizId)},
		fieldLikeCnt, -1).Err()
}

func (r *RedisInteractiveCache) Get(ctx context.Context,
	biz string, bizId int64) (domain.Interactive, error) {
	// 直接使用 HMGet，即便缓存中没有对应的 key，也不会返回 error
	data, err := r.client.HGetAll(ctx, r.key(biz, bizId)).Result()
	if err != nil {
		return domain.Interactive{}, err
	}

	if len(data) == 0 {
		// 缓存不存在
		return domain.Interactive{}, ErrKeyNotExist
	}

	// 理论上来说，这里不可能有 error
	collectCnt, _ := strconv.ParseInt(data[fieldCollectCnt], 10, 64)
	likeCnt, _ := strconv.ParseInt(data[fieldLikeCnt], 10, 64)
	readCnt, _ := strconv.ParseInt(data[fieldReadCnt], 10, 64)

	return domain.Interactive{
		// 懒惰的写法
		CollectCnt: collectCnt,
		LikeCnt:    likeCnt,
		ReadCnt:    readCnt,
	}, err
}

func (r *RedisInteractiveCache) Set(ctx context.Context, biz string, bizId int64, intr domain.Interactive) error {
	//这里只缓存通用部分的
	key := r.key(biz, bizId)
	err := r.client.HMSet(ctx, key,
		fieldLikeCnt, intr.LikeCnt,
		fieldCollectCnt, intr.CollectCnt,
		fieldReadCnt, intr.ReadCnt).Err()
	if err != nil {
		return err
	}
	return r.client.Expire(ctx, key, time.Minute*15).Err()
}

func (r *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}

func NewRedisInteractiveCache(client redis.Cmdable) InteractiveCache {
	return &RedisInteractiveCache{
		client: client,
		tracer: otel.GetTracerProvider().Tracer("redis-cache"),
	}
}
