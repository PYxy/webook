package article

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/demdxx/gocast/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type Producer interface {
	ProduceReadEvent(ctx context.Context, evt ReadEvent) error
}

type KafkaProducer struct {
	producer sarama.SyncProducer
	summary  *prometheus.SummaryVec
}

// ProduceReadEvent 如果你有复杂的重试逻辑，就用装饰器
// 你认为你的重试逻辑很简单，你就放这里
func (k *KafkaProducer) ProduceReadEvent(ctx context.Context, evt ReadEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	start := time.Now()
	defer func() {
		k.summary.WithLabelValues("read_article").Observe(gocast.Float64(time.Now().Sub(start).Milliseconds()))
	}()
	_, _, err = k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: "read_article",
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func NewKafkaProducer(pc sarama.SyncProducer, name string) Producer {
	tmpSummary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "webook",
		Subsystem: "article_Producer",
		Name:      name,
		Help:      "记录生产者 消息存储的时间",
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.9:   0.01,
			0.99:  0.005,
			0.999: 0.0001,
		},
	}, []string{"topic"})
	prometheus.MustRegister(tmpSummary)
	return &KafkaProducer{
		producer: pc,
		summary:  tmpSummary,
	}
}

type ReadEvent struct {
	Uid int64
	Aid int64
}
