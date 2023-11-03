package sarama

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestConsumer(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	version, err := sarama.ParseKafkaVersion("2.1.1")
	cfg.Version = version
	// 正常来说，一个消费者都是归属于一个消费者的组的
	// 消费者组就是你的业务
	consumer, err := sarama.NewConsumerGroup(addrs,
		"test_group", cfg)
	require.NoError(t, err)

	// 带超时的 context
	start := time.Now()
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	time.AfterFunc(time.Second*30, func() {
		cancel()
	})
	err = consumer.Consume(ctx,
		[]string{"test_topic"}, testConsumerGroupHandler{})
	// 你消费结束，就会到这里
	t.Log(err, time.Since(start).String())
}

type testConsumerGroupHandler struct {
}

func (t testConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	// topic => 偏移量
	//partitions := session.Claims()["test_topic"]

	//for _, part := range partitions {
	//	log.Println("partition:", part)
	//	//session.ResetOffset("test_topic", part,
	//	//	sarama.OffsetOldest, "")
	//	//session.ResetOffset("test_topic", part,
	//	//	400, "")
	//	//session.ResetOffset("test_topic", part,
	//	//	sarama.OffsetNewest, "")
	//}
	//session.ResetOffset("test_topic", 0,
	//	sarama.OffsetOldest, "")
	log.Println("setUp over")

	return nil
}

func (t testConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	log.Println("CleanUp")
	session.ResetOffset("test_topic", 0,
		sarama.OffsetOldest, "")
	return nil
}

func (t testConsumerGroupHandler) ConsumeClaim(
// 代表的是你和Kafka 的会话（从建立连接到连接彻底断掉的那一段时间）
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	//for msg := range msgs {
	//	m1 := msg
	//	go func() {
	//		// 消费msg
	//		log.Println(string(m1.Value))
	//		session.MarkMessage(m1, "")
	//	}()
	//}
	const batchSize = 10
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		var eg errgroup.Group
		var last *sarama.ConsumerMessage
		var offset int64
		done := false
		//log.Println("循环")
		for i := 0; i < batchSize && !done; i++ {
			select {
			case <-ctx.Done():
				//goto label1
				// 这边代表超时了
				done = true
			case msg, ok := <-msgs:
				//log.Println("消息进来了。。。", ok)
				if !ok {
					cancel()
					// 代表消费者被关闭了
					return nil
				}

				last = msg
				offset = msg.Offset
				eg.Go(func() error {
					// 我就在这里消费
					//time.Sleep(time.Second)
					// 你在这里重试
					log.Println(string(msg.Value))
					return nil
				})
			}
		}
		cancel()
		err := eg.Wait()
		if err != nil {
			log.Println("出错了:", err)
			// 这边能怎么办？
			// 记录日志
			continue
		}
		// 就这样
		if last != nil {
			log.Println("提交？？最大offset:", offset)
			session.MarkMessage(last, "")
		}
	}
	//
	//label1:
	//	println("hellp")
}

func (t testConsumerGroupHandler) ConsumeClaimV1(
// 代表的是你和Kafka 的会话（从建立连接到连接彻底断掉的那一段时间）
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		//var bizMsg MyBizMsg
		//err := json.Unmarshal(msg.Value, &bizMsg)
		//if err != nil {
		//	// 这就是消费消息出错
		//	// 大多数时候就是重试
		//	// 记录日志
		//	continue
		//}
		log.Println(string(msg.Value))
		session.MarkMessage(msg, "")
	}
	// 什么情况下会到这里
	// msgs 被人关了，也就是要退出消费逻辑
	return nil
}

type MyBizMsg struct {
	Name string
}

// 返回只读的 channel
func ChannelV1() <-chan struct{} {
	panic("implement me")
}

// 返回可读可写 channel
func ChannelV2() chan struct{} {
	panic("implement me")
}

// 返回只写 channel
func ChannelV3() chan<- struct{} {
	panic("implement me")
}

/*
创建一个3分区1副本名为test的topic，必须指定分区数 --partitions 和副本数--replication-factor，其中副本数量不能超过kafka节点（broker）数量
I have no name!@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --topic ljy --partitions 3 --replication-factor 1 --create
Created topic ljy.

#获取多有的topic
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --list
__consumer_offsets
ljy
test_topic

#往topic 写入消息


#删除topic
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --topic ljy --delete


#只获取 offset为0 的一条消息
--max-messages获取多少条  不写就从offset 开始一直watch
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --offset 1 --partition 0 --topic test_topic --max-messages 1
Hello, 这是一条消息 A
Processed a total of 1 messages

#获取 topic 的详细信息
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --describe
Topic: test_topic	TopicId: GC3S4zrpRGi5i5UwmtU_wA	PartitionCount: 1	ReplicationFactor: 1	Configs:
	Topic: test_topic	Partition: 0	Leader: 0	Replicas: 0	Isr: 0

#增加topic 分区数增加到个分去数
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 -alter --partitions 2 --topic test_topic
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --describe
Topic: test_topic	TopicId: GC3S4zrpRGi5i5UwmtU_wA	PartitionCount: 2	ReplicationFactor: 1	Configs:
	Topic: test_topic	Partition: 0	Leader: 0	Replicas: 0	Isr: 0
	Topic: test_topic	Partition: 1	Leader: 0	Replicas: 0	Isr: 0


#消费者组
设置从最开始的滴地方笑消费
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-consumer-groups.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --group t99  --reset-offsets --to-earliest --execute

GROUP                          TOPIC                          PARTITION  NEW-OFFSET
t99                            test_topic                     0          0
t99                            test_topic                     1          0

#开两个窗口 模拟t99消费者组里面有2个消费者
#从头开始消费
./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --group t99 --from-beginning


#1
I have no name!@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --gro
Hello, 这是一条消息 A
Hello, 这是一条消息 A

#2
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --group t99
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A


#退出2个消费者再重新启动就不会有消息 消费 因为已经提交过了

#查看消费者组的消费情况(积压情况)
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-consumer-groups.sh -bootstrap-server 127.0.0.1:9092 --group t99 --describe

GROUP           TOPIC           PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG             CONSUMER-ID                                           HOST            CLIENT-ID
t99             test_topic      0          7               7               0               console-consumer-353f5455-6982-48a9-839c-7a6c09fec977 /172.23.0.2     console-consumer
t99             test_topic      1          2               2               0               console-consumer-a7e41f9e-93c3-4817-82f8-9065364d0513 /172.23.0.2     console-consume


#查看 各个分区的offset

@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-get-offsets.sh -bootstrap-server 127.0.0.1:9092 --topic test_topic --partitions 0
test_topic:0:7
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-get-offsets.sh -bootstrap-server 127.0.0.1:9092 --topic test_topic --partitions 1
test_topic:1:2

#重新设置分区的offset
--to-datetime <String: datetime>        Reset offsets to offset from datetime.
                                          Format: 'YYYY-MM-DDTHH:mm:SS.sss'
--to-earliest                           Reset offsets to earliest offset.
--to-latest                             Reset offsets to latest offset.
--to-offset <Long: offset>              Reset offsets to a specific offset.


Error: Assignments can only be reset if the group 't99' is inactive, but the current state is Stable.
#x需要退出 t99消费者组才可以设置

GROUP                          TOPIC                          PARTITION  NEW-OFFSET

#将topic 的所有分区offset同时设定为 0
I have no name!@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-consumer-groups.sh --bootstrap-server 127.0.0.1:9092 --group t99 --topic test_topic --reset-offsets --to-offset 0 --execute

GROUP                          TOPIC                          PARTITION  NEW-OFFSET
t99                            test_topic                     0          0
t99                            test_topic                     1          0


#设置指定分区的偏移量   设定分区1的偏移量为1
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-consumer-groups.sh --bootstrap-server 127.0.0.1:9092 --group t99 --topic test_topic:1 --reset-offsets --to-offset 1 --execute
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-consumer-groups.sh -bootstrap-server 127.0.0.1:9092 --group t99 --describe

Consumer group 't99' has no active members.

GROUP           TOPIC           PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG             CONSUMER-ID     HOST            CLIENT-ID
t99             test_topic      0          0               7               7               -               -               -
t99             test_topic      1          1               2               1               -               -               -


#再启动消费者
#1
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --group t99 --from-beginning
Hello, 这是一条消息 A

#2
@bb9b8f0ccffc:/opt/bitnami/kafka/bin$ ./kafka-console-consumer.sh --bootstrap-server 127.0.0.1:9092 --topic test_topic --group t99
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A
Hello, 这是一条消息 A

*/
