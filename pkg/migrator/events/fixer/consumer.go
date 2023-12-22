package fixer

import (
	"context"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/events"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/fixer"
	"gitee.com/geekbang/basic-go/webook/pkg/saramax"
)

type Consumer[T migrator.Entity] struct {
	client   sarama.Client
	l        logger.LoggerV1
	srcFirst *fixer.OverrideFixer[T]
	dstFirst *fixer.OverrideFixer[T]
	topic    string
}

func NewConsumer[T migrator.Entity](
	client sarama.Client,
	l logger.LoggerV1,
	topic string,
	src *gorm.DB,
	dst *gorm.DB) (*Consumer[T], error) {
	srcFirst, err := fixer.NewOverrideFixer[T](src, dst)
	if err != nil {
		return nil, err
	}
	dstFirst, err := fixer.NewOverrideFixer[T](dst, src)
	if err != nil {
		return nil, err
	}
	return &Consumer[T]{
		client:   client,
		l:        l,
		srcFirst: srcFirst,
		dstFirst: dstFirst,
		topic:    topic,
	}, nil
}

// Start 这边就是自己启动 goroutine 了
func (r *Consumer[T]) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("migrator-fix",
		r.client)
	if err != nil {
		return err
	}
	go func() {
		err := cg.Consume(context.Background(),
			[]string{r.topic},
			saramax.NewHandler[events.InconsistentEvent](r.l, r.Consume))
		if err != nil {
			r.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()
	return err
}

func (r *Consumer[T]) Consume(msg *sarama.ConsumerMessage, t events.InconsistentEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	switch t.Direction {
	case "SRC":
		return r.srcFirst.Fix(ctx, t.ID)
	case "DST":
		return r.dstFirst.Fix(ctx, t.ID)
	}
	return errors.New("未知的校验方向")
}

func (r *Consumer[T]) Startv2() error {
	cg, err := sarama.NewConsumerGroupFromClient("migrator-fix",
		r.client)
	if err != nil {
		return err
	}
	go func() {
		err := cg.Consume(context.Background(),
			[]string{r.topic},
			saramax.NewBatchHandler[events.InconsistentEvent](r.l, r.ConsumeBatches))
		if err != nil {
			r.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()
	return err
}

// 消费者批量处理数据
func (r *Consumer[T]) ConsumeBatches(msg []*sarama.ConsumerMessage, t []events.InconsistentEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var tmpMap map[string][]int64
	tmpMap = make(map[string][]int64, 3)
	for _, event := range t {
		_, ok := tmpMap[event.Direction]
		if !ok {
			tmpMap[event.Direction] = make([]int64, 20)
		}
		tmpMap[event.Direction] = append(tmpMap[event.Direction], event.ID)
	}
	//这里的value 可以不用去重  因为后面的sql 操作已经做了id 去重操作
	for directio, idSlice := range tmpMap {
		switch directio {
		case "SRC":
			return r.srcFirst.FixInBatches(ctx, idSlice)
		case "DST":
			return r.dstFirst.FixInBatches(ctx, idSlice)
		}
		return errors.New("未知的校验方向")
	}

	return nil
}
