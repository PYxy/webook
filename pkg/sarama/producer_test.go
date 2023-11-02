package sarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var addrs = []string{"xxxxx:9094"}

// TestSyncProducer 同步提提交
func TestSyncProducer(t *testing.T) {
	cfg := sarama.NewConfig()

	//第二种 创建客户端方式
	//client, err := sarama.NewClient(addrs, cfg)
	//producer := sarama.NewSyncProducerFromClient(client)

	//cfg := new(sarama.Config)
	//cfg := sarama.Config{
	//	Admin: struct {
	//		Max struct {
	//			Max     int
	//			Timeout time.Duration
	//		}
	//		Timeout time.Duration
	//	}{Max: , Timeout: },
	//}
	cfg.Producer.Return.Successes = true
	//默认使用hash
	cfg.Producer.Partitioner = sarama.NewHashPartitioner
	//cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	//cfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	//cfg.Producer.Partitioner = sarama.NewManualPartitioner
	//cfg.Producer.Partitioner = sarama.NewCustomPartitioner()
	//cfg.Producer.Partitioner = sarama.NewCustomHashPartitioner(func() hash.Hash32 {
	//
	//})
	//
	//cfg.Producer.Partitioner = sarama.NewHashPartitioner
	producer, err := sarama.NewSyncProducer(addrs, cfg)
	assert.NoError(t, err)
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "test_topic",
		Key:   sarama.StringEncoder("oid-123"),
		// 消息数据本体
		// 转 JSON
		// protobuf
		Value: sarama.StringEncoder("Hello, 这是一条消息 A"),
		// 会在生产者和消费者之间传递
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("trace_id"),
				Value: []byte("123456"),
			},
		},
		// 只作用于发送过程
		Metadata: "这是metadata",
	})
	assert.NoError(t, err)
}

// TestSyncProducer 异步提提交
func TestAsyncProducer(t *testing.T) {
	cfg := sarama.NewConfig()
	//表示关注发送消息是否成功 或者异常
	cfg.Producer.Return.Errors = true
	cfg.Producer.Return.Successes = true
	producer, err := sarama.NewAsyncProducer(addrs, cfg)
	require.NoError(t, err)
	msgCh := producer.Input()

	go func() {
		for {
			msg := &sarama.ProducerMessage{
				Topic: "test_topic",
				Key:   sarama.StringEncoder("oid-123"),
				Value: sarama.StringEncoder("Hello, 这是一条消息 A"),
				Headers: []sarama.RecordHeader{
					{
						Key:   []byte("trace_id"),
						Value: []byte("123456"),
					},
				},
				Metadata: "这是metadata",
			}
			select {
			case msgCh <- msg:
				//default:

			}

		}
	}()

	errCh := producer.Errors()
	succCh := producer.Successes()

	for {
		// 如果两个情况都没发生，就会阻塞
		select {
		case err := <-errCh:
			t.Log("发送出了问题", err.Err)
		case <-succCh:
			t.Log("发送成功")
		}
	}
}

// 需要实现 sarama.StringEncoder接口
type JSONEncoder struct {
	Data any
}

//func (s JSONEncoder) Encode() ([]byte, error) {
//	return []byte(s), nil
//}
//
//func (s JSONEncoder) Length() int {
//	return len(s)
//}
