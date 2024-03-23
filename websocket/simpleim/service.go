package simpleim

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"strconv"
)

// IMService 代表了我们后端服务
type IMService struct {
	producer sarama.SyncProducer
}

func (s *IMService) Receive(ctx context.Context, sender int64, msg Message) error {
	// 这边就是业务的大头

	// 审核，如果不通过，就拒绝，也在这个地方，一定是同步的，而且最先执行

	// 我要先找到接收者..
	receivers := s.findMembers()

	// 同步数据过去给搜索，你可以在这里做，也可以借助消费 eventName 来
	// 消息记录存储，也可以在这里做。一般存一条

	for _, receiver := range receivers {
		if receiver == sender {
			// 你自己就不要转发了
			// 但是，如果你有多端同步，你还得转发
			continue
		}
		// 一个个转发
		// 你要注意一点
		// 正常来说，这边你可以考虑顺序问题了
		// 这边。你可以考虑改批量接口
		event, _ := json.Marshal(Event{Msg: msg, Receiver: receiver})
		_, _, err := s.producer.SendMessage(&sarama.ProducerMessage{
			Topic: eventName,
			// 可以考虑，在初始话 producer 的时候，使用哈希类的 partition 选取策略
			Key:   sarama.StringEncoder(strconv.FormatInt(receiver, 10)),
			Value: sarama.ByteEncoder(event),
		})
		if err != nil {
			// 记录日志 + 重试
			continue
		}
	}
	return nil
}

// 这里模拟根据 cid，也就是聊天 ID 来查找参与了该聊天的成员
func (s *IMService) findMembers() []int64 {
	// 固定返回 1，2，3，4
	// 正常来说，你这里要去找你的聊天的成员
	return []int64{1, 2, 3, 4}
}
