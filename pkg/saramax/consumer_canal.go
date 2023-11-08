package saramax

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/golang/protobuf/proto"
	pb "github.com/withlin/canal-go/protocol"
	pbe "github.com/withlin/canal-go/protocol/entry"

	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type CanalHandler[T any] struct {
	l  logger.LoggerV1
	fn func(msg *sarama.ConsumerMessage, t T) error
}

func NewCanalHandler[T any](l logger.LoggerV1, fn func(msg *sarama.ConsumerMessage, t T) error) *CanalHandler[T] {
	return &CanalHandler[T]{
		l:  l,
		fn: fn,
	}
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
	for message := range msgs {
		session.MarkMessage(message, "")

		mes, err := pb.Decode(message.Value, false)
		if err != nil {
			h.l.Warn("message pb.Decode解析信息失败", logger.Error(err),
				logger.Int64("offset", message.Offset),
				logger.Int64("partition", int64(message.Partition)),
				logger.String("topic", message.Topic))

			continue
		}
		err = printEntry(mes.Entries)
		if err != nil {
			h.l.Warn("message pb.Decode解析信息失败", logger.Error(err),
				logger.Int64("offset", message.Offset),
				logger.Int64("partition", int64(message.Partition)),
				logger.String("topic", message.Topic))

			continue
		}

	}
	return nil
}

func printEntry(entrys []pbe.Entry) error {

	for _, entry := range entrys {
		if entry.GetEntryType() == pbe.EntryType_TRANSACTIONBEGIN || entry.GetEntryType() == pbe.EntryType_TRANSACTIONEND {
			continue
		}
		rowChange := new(pbe.RowChange)

		err := proto.Unmarshal(entry.GetStoreValue(), rowChange)
		if err != nil {
			return err

		}
		if rowChange != nil {
			eventType := rowChange.GetEventType()
			//header := entry.GetHeader()
			//fmt.Println(fmt.Sprintf("================> binlog[%s : %d],name[%s,%s], eventType: %s", header.GetLogfileName(), header.GetLogfileOffset(), header.GetSchemaName(), header.GetTableName(), header.GetEventType()))

			for _, rowData := range rowChange.GetRowDatas() {
				if eventType == pbe.EventType_DELETE {
					printColumn(rowData.GetBeforeColumns())
				} else if eventType == pbe.EventType_INSERT {
					printColumn(rowData.GetAfterColumns())
				} else {
					fmt.Println("-------> before")
					printColumn(rowData.GetBeforeColumns())
					fmt.Println("-------> after")
					printColumn(rowData.GetAfterColumns())
				}
			}
		}
	}
	return nil
}

func printColumn(columns []*pbe.Column) {
	for _, col := range columns {
		//fmt.Println("字段名 \t  值 \t  值类型 发生更改 \t")
		fmt.Println(fmt.Sprintf("字段名：%s\t是不是主键：%t\t值：%s\tchange：%t\t", col.GetName(), col.IsKey, col.GetValue(), col.GetUpdated()))
		//fmt.Println(fmt.Sprintf("%s : %s   %t", col.GetName(), col.GetValue(), col.GetUpdated()))
	}
}
