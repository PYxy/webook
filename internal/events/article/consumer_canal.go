package article

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/queue"
	"gitee.com/geekbang/basic-go/webook/pkg/saramax"
)

type CANALEVENT struct {
	Id       int64  `mapstructure:"id"`
	BizId    int64  `mapstructure:"biz_id"`
	Biz      string `mapstructure:"biz"`
	Like_cnt int64  `mapstructure:"like_cnt"`
	Read_cnt int64  `mapstructure:"read_cnt"`
}

func (c CANALEVENT) ToDomain() domain.TopInteractive {
	return domain.TopInteractive{
		Id:      c.Id,
		BizId:   c.BizId,
		Biz:     c.Biz,
		ReadCnt: c.Read_cnt,
		LikeCnt: c.Like_cnt,
	}
}

func (c CANALEVENT) Key() string {
	return fmt.Sprintf("%v:%v:%v", c.Id, c.BizId, c.Biz)
}

type TopNConsumer struct {
	client    sarama.Client
	repo      repository.InteractiveRepository
	count     int64
	localDate map[string]domain.TopInteractive

	l     logger.LoggerV1
	queue *queue.PriorityQueue[domain.TopInteractive]
}

func (c *TopNConsumer) Consume(event []CANALEVENT) error {
	fmt.Println("需要更新的事件:", event)
	fmt.Println("原始数据1:", c.localDate)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//要先判断 这个key 是不是已经在里面了
	//处理CANALEVENT  这里其实是一个 切片更好
	notExistEvent := make([]CANALEVENT, 0, 10)
	var needToSort bool
	for _, e := range event {
		key := e.Key()
		_, ok := c.localDate[key]
		if !ok {
			notExistEvent = append(notExistEvent, e)
			continue
		}

		c.localDate[key] = e.ToDomain()
		needToSort = true

	}

	fmt.Println("原始数据2:", c.queue.RawData())
	//原有的排名中的人 有更新的先更新
	if needToSort {
		newQueue := queue.NewPriorityQueue[domain.TopInteractive](-1, func(src domain.TopInteractive, dst domain.TopInteractive) int {
			if src.LikeCnt < dst.LikeCnt {
				return -1
			} else if src.LikeCnt == dst.LikeCnt {
				return 0
			} else {
				return 1
			}
		})
		for _, val := range c.localDate {
			_ = newQueue.Enqueue(val)
		}
		c.queue = newQueue

	}

	if len(notExistEvent) >= 0 {
		for _, e := range notExistEvent {
			//直接拿第一个就好了
			val, _ := c.queue.Peek()
			if val.LikeCnt <= e.Like_cnt {
				//删除map 中的
				delete(c.localDate, e.Key())
				//出头
				_, _ = c.queue.Dequeue()
				//从新排序
				_ = c.queue.Enqueue(e.ToDomain())
			}
		}
	}

	//return r.repo.IncrReadCnt(ctx, "article", t.Aid)
	return c.repo.SetTopN(ctx, c.queue.RawData())
	//直接在小顶推中操作元素  或者在上层做一个聚合操作
}

func (t TopNConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("TopN",
		t.client)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	res, err := t.repo.GetTopN(ctx)
	fmt.Println("查询到的结果：", res)
	if err != nil {
		panic(err)
	}
	for _, topInteractice := range res {
		t.localDate[topInteractice.Key()] = topInteractice
		//加入到小顶推中
		t.queue.Enqueue(topInteractice)
	}

	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	err = t.repo.SetTopN(ctx, res)
	if err != nil {
		panic(fmt.Sprintf("设置top N(10)失败:%v", err))
	}
	cancel()
	//异步统计 redis 中的数据
	//异步消费kafka 中 cancal 中的消息
	go func() {
		err := cg.Consume(context.Background(),
			[]string{"webook_interactives"}, //topic
			//实现那3个方法的对象
			saramax.NewCanalHandler[CANALEVENT](t.l, t.Consume))
		if err != nil {
			t.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()

	return err
}

func NewTopNConsumer(repo repository.InteractiveRepository, client sarama.Client, count int64, l logger.LoggerV1, data *queue.PriorityQueue[domain.TopInteractive]) *TopNConsumer {
	return &TopNConsumer{
		client:    client,
		count:     count,
		repo:      repo,
		localDate: map[string]domain.TopInteractive{},
		l:         l,
		queue:     data,
	}
}
