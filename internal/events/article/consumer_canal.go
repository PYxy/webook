package article

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/saramax"
)

type CANALEVENT struct {
	id    int64
	bizId string
	biz   string
	cnt   int
}

type TopNConsumer struct {
	client sarama.Client
	repo   repository.InteractiveRepository
	count  int64
	cmd    redis.Cmdable
	l      logger.LoggerV1
}

func (t *TopNConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive",
		t.client)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	res, err := t.repo.GetTopN(ctx)
	if err != nil {
		panic(err)
	}
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	err = t.SetTopN(ctx, res)
	if err != nil {
		panic(fmt.Sprintf("设置topM失败:%v", err))
	}
	cancel()
	//异步统计 redis 中的数据
	//异步消费kafka 中 cancal 中的消息
	go func() {
		err := cg.Consume(context.Background(),
			[]string{"interactive"}, //topic
			//实现那3个方法的对象
			saramax.NewCanalHandler[CANALEVENT](t.l, t.Consume))
		if err != nil {
			t.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()

	return err
}

func (t *TopNConsumer) Consume(msg *sarama.ConsumerMessage, t2 CANALEVENT) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return r.repo.IncrReadCnt(ctx, "article", t.Aid)
}

func (t *TopNConsumer) SetTopN(ctx context.Context, res []domain.TopInteractive) error {
	return nil
}

func NewTopNConsumer(client sarama.Client, count int64, cmd redis.Cmdable, l logger.LoggerV1) *TopNConsumer {
	return &TopNConsumer{
		client: client,
		count:  count,
		cmd:    cmd,
		l:      l,
	}
}
