package saramax

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/demdxx/gocast/v2"
	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/client_golang/prometheus"
	pb "github.com/withlin/canal-go/protocol"
	pbe "github.com/withlin/canal-go/protocol/entry"

	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type CanalHandler[T any] struct {
	l       logger.LoggerV1
	fn      func(t []T) error
	gauge   *prometheus.GaugeVec
	counter *prometheus.CounterVec
}

func NewCanalHandler[T any](l logger.LoggerV1, fn func(t []T) error, name string) *CanalHandler[T] {
	labels := []string{"topic", "partition"}
	tmpGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "webook",
		Subsystem:   "article_consumer",
		Name:        name,
		Help:        "统计 kafka 分区的offset(看下有没有数据倾斜)",
		ConstLabels: nil,
	}, labels)
	tmpCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "webook",
		Subsystem: "article_consumer",
		Name:      name,
		Help:      "统计 从kafka 中每个消息消费的速度",
	}, labels)
	ch := &CanalHandler[T]{
		l:       l,
		fn:      fn,
		gauge:   tmpGauge,
		counter: tmpCounter,
	}

	prometheus.MustRegister([]prometheus.Collector{tmpGauge, tmpCounter}...)

	return ch
}

func (h CanalHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h CanalHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 可以使用装饰器封装重试
func (h CanalHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	const batchSize = 10
	for {
		resSlice := make([]T, 0, batchSize)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		offsetSLice := make([]*sarama.ConsumerMessage, 0, batchSize)
		done := false
		for i := 0; i < batchSize; i++ {
			select {
			case <-ctx.Done():
				done = true
				//超时结束
			case message, ok := <-msgs:
				if !ok {
					cancel()
					return nil
				}
				offsetSLice = append(offsetSLice, message)
				mes, err := pb.Decode(message.Value, false)
				if err != nil {
					h.l.Warn("message pb.Decode解析信息失败", logger.Error(err),
						logger.Int64("offset", message.Offset),
						logger.Int64("partition", int64(message.Partition)),
						logger.String("topic", message.Topic))

					continue
				}
				res, err := printEntry[T](mes.Entries)
				if err != nil {
					h.counter.WithLabelValues(message.Topic, gocast.Str(message.Partition)).Inc()
					h.l.Warn("解析信息失败", logger.Error(err),
						logger.Int64("offset", message.Offset),
						logger.Int64("partition", int64(message.Partition)),
						logger.String("topic", message.Topic))

					continue
				}
				resSlice = append(resSlice, res)

			}
			if done {
				break
			}
		}
		//多个分区只能这样提交
		//单个分区 只需要提交最后获取的一个消息就行
		for _, commit := range offsetSLice {
			session.MarkMessage(commit, "")
		}
		if len(resSlice) != 0 {
			fmt.Println("处理结果:", h.fn(resSlice))
		}

	}
	//for message := range msgs {
	//	session.MarkMessage(message, "")
	//
	//	mes, err := pb.Decode(message.Value, false)
	//	if err != nil {
	//		h.l.Warn("message pb.Decode解析信息失败", logger.Error(err),
	//			logger.Int64("offset", message.Offset),
	//			logger.Int64("partition", int64(message.Partition)),
	//			logger.String("topic", message.Topic))
	//
	//		continue
	//	}
	//	res, err := printEntry[T](mes.Entries)
	//	if err != nil {
	//		h.l.Warn("message pb.Decode解析信息失败", logger.Error(err),
	//			logger.Int64("offset", message.Offset),
	//			logger.Int64("partition", int64(message.Partition)),
	//			logger.String("topic", message.Topic))
	//
	//		continue
	//	}
	//	fmt.Println("处理结果:", h.fn(message, res))
	//
	//}

}

func printEntry[T any](entrys []pbe.Entry) (T, error) {
	var t T
	for _, entry := range entrys {
		if entry.GetEntryType() == pbe.EntryType_TRANSACTIONBEGIN || entry.GetEntryType() == pbe.EntryType_TRANSACTIONEND {
			continue
		}
		rowChange := new(pbe.RowChange)

		err := proto.Unmarshal(entry.GetStoreValue(), rowChange)
		if err != nil {
			return t, err

		}
		if rowChange != nil {
			eventType := rowChange.GetEventType()
			//header := entry.GetHeader()
			//fmt.Println(fmt.Sprintf("================> binlog[%s : %d],name[%s,%s], eventType: %s", header.GetLogfileName(), header.GetLogfileOffset(), header.GetSchemaName(), header.GetTableName(), header.GetEventType()))

			for _, rowData := range rowChange.GetRowDatas() {
				if eventType == pbe.EventType_DELETE {
					//printColumn(rowData.GetBeforeColumns())

				} else if eventType == pbe.EventType_INSERT {
					//printColumn(rowData.GetAfterColumns())
					return printColumn[T](rowData.GetAfterColumns())
				} else {
					//fmt.Println("-------> before")
					//printColumn(rowData.GetBeforeColumns())
					//fmt.Println("-------> after")
					//printColumn(rowData.GetAfterColumns())
					return printColumn[T](rowData.GetAfterColumns())
				}
			}
		}
	}
	return t, errors.New("no change")
}

func printColumn[T any](columns []*pbe.Column) (res T, err error) {
	var t T
	tmpMap := make(map[string]string, len(columns))
	for _, col := range columns {
		//fmt.Println("字段名 \t  值 \t  值类型 发生更改 \t")
		//fmt.Println(fmt.Sprintf("字段名：%s\t是不是主键：%t\t值：%s\tchange：%t\t", col.GetName(), col.IsKey, col.GetValue(), col.GetUpdated()))
		tmpMap[col.GetName()] = col.GetValue()
		//fmt.Println(fmt.Sprintf("%s : %s   %t", col.GetName(), col.GetValue(), col.GetUpdated()))
	}
	fmt.Println("原始数据:", tmpMap)
	if err = mapstructure.WeakDecode(tmpMap, &t); err != nil {
		return t, err
	}
	fmt.Println("转化成功:", t)
	return t, nil
}
