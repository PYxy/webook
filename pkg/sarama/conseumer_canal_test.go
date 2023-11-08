// canal + kafka
package sarama

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/golang/protobuf/proto"
	pb "github.com/withlin/canal-go/protocol"
	pbe "github.com/withlin/canal-go/protocol/entry"
)

func TestConsumer2(t *testing.T) {
	newKafkaConsumer()
}

// 开始创建kafka订阅者
func newKafkaConsumer() {
	/**
	    group:
	   设置订阅者群 如果多个订阅者group一样，则随机挑一个进行消费，当然也可以设置轮训，在设置里面修改；
	   若多个订阅者的group不同，则一旦发布者发布消息，所有订阅者都会订阅到同样的消息；
	  topics:
	   逻辑分区必须与发布者相同，还是用安彦飞，不然找不到内容咯
	   当然订阅者是可以订阅多个逻辑分区的，只不过因为演示方便我写了一个，你可以用英文逗号分割在这里写多个
	*/
	var (
		group  = "Consumer2"
		topics = "webook_interactives"
	)
	var address = []string{"120.132.118.90:9094"}
	log.Println("Starting a new Sarama consumer")
	//配置订阅者
	config := sarama.NewConfig()
	//配置偏移量
	config.Consumer.Offsets.Initial = sarama.OffsetOldest //初始从最新的offset开始(未被消费的数据)
	//config.Consumer.Offsets.Initial = sarama.OffsetOldest //初始从旧的offset开始
	//config.Consumer.Offsets.Initia =
	//config.Consumer.Offsets.AutoCommit.Enable = true. // 自动提交
	//config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second // 间隔
	//config.Consumer.Offsets.Retry.Max = 3
	//开始创建订阅者
	consumer := Consumer{
		ready: make(chan bool),
	}
	//创建一个上下文对象，实际项目中也一定不要设置超时（当然，按你项目需求，我是没见过有项目需求要多少时间后取消订阅的）
	ctx, cancel := context.WithCancel(context.Background())
	//创建订阅者群，集群地址发布者代码里已定义
	client, err := sarama.NewConsumerGroup(address, group, config)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	//创建同步组
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			/**
			  官方说：`订阅者`应该在无限循环内调用
			  当`发布者`发生变化时
			  需要重新创建`订阅者`会话以获得新的声明

			  所以这里把订阅者放在了循环体内
			*/
			if err := client.Consume(ctx, strings.Split(topics, ","), &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// 检查上下文是否被取消，收到取消信号应当立刻在本协程中取消循环
			if ctx.Err() != nil {
				return
			}
			//获取订阅者准备就绪信号
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // 获取到了订阅者准备就绪信号后打印下面的话
	log.Println("Sarama consumer up and running!...")

	//golang优雅退出的信号通道创建
	sigterm := make(chan os.Signal, 1)
	//golang优雅退出的信号获取
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	//创建选择器，如果不是上下文取消或者用户ctrl+c这种系统级退出，则就不向下执行了
	select {
	case <-ctx.Done():
		log.Println("terminating: context cancelled")
	case <-sigterm:
		log.Println("terminating: via signal")
	}
	//取消上下文
	cancel()
	wg.Wait()
	//关闭客户端
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

// 重写订阅者，并重写订阅者的所有方法
type Consumer struct {
	ready chan bool
}

// Setup方法在新会话开始时运行的，然后才使用声明
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	fmt.Println("SETUP")
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// 一旦所有的订阅者协程都退出，Cleaup方法将在会话结束时运行
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	fmt.Println("Cleanup")
	return nil
}

// 订阅者在会话中消费消息，并标记当前消息已经被消费。
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		//信息确认
		session.MarkMessage(message, "")
		//普通信息
		//log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s, partition= %d, offset= %d", string(message.Value), message.Timestamp, message.Topic, message.Partition, message.Offset)

		//解析canal 信息
		mes, err := pb.Decode(message.Value, false)
		fmt.Println("是不是异常:", err)
		fmt.Println(mes)
		batchId := mes.Id
		if batchId == -1 || len(mes.Entries) <= 0 {
			//fmt.Println(".....")
			time.Sleep(300 * time.Millisecond)
			//fmt.Println("===没有数据了===")
			continue
		}

		printEntry(mes.Entries)
		//err = proto.Unmarshal(message.Value, rowChange)
		//checkError(err)
		//fmt.Println(rowChange.String())
		//fmt.Println(rowChange.RowDatas)
		//if rowChange != nil {
		//  eventType := rowChange.GetEventType()
		//  fmt.Printf("%T \n", eventType)
		//  for _, rowData := range rowChange.GetRowDatas() {
		//    fmt.Println(rowData)
		//    if eventType == pbe.EventType_DELETE {
		//      printColumn(rowData.GetBeforeColumns())
		//    } else if eventType == pbe.EventType_INSERT {
		//      fmt.Println("插入")
		//      printColumn(rowData.GetAfterColumns())
		//    } else {
		//      fmt.Println("-------> before")
		//      printColumn(rowData.GetBeforeColumns())
		//      fmt.Println("-------> after")
		//      printColumn(rowData.GetAfterColumns())
		//    }
		//  }
		//}
	}

	return nil
}

//处理canal 信息

func printEntry(entrys []pbe.Entry) {

	for _, entry := range entrys {
		if entry.GetEntryType() == pbe.EntryType_TRANSACTIONBEGIN || entry.GetEntryType() == pbe.EntryType_TRANSACTIONEND {
			continue
		}
		rowChange := new(pbe.RowChange)

		err := proto.Unmarshal(entry.GetStoreValue(), rowChange)
		checkError(err)
		if rowChange != nil {
			eventType := rowChange.GetEventType()
			header := entry.GetHeader()
			fmt.Println(fmt.Sprintf("================> binlog[%s : %d],name[%s,%s], eventType: %s", header.GetLogfileName(), header.GetLogfileOffset(), header.GetSchemaName(), header.GetTableName(), header.GetEventType()))

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
}

func printColumn(columns []*pbe.Column) {
	for _, col := range columns {
		//fmt.Println("字段名 \t  值 \t  值类型 发生更改 \t")
		fmt.Println(fmt.Sprintf("字段名：%s\t是不是主键：%t\t值：%s\tchange：%t\t", col.GetName(), col.IsKey, col.GetValue(), col.GetUpdated()))
		//fmt.Println(fmt.Sprintf("%s : %s   %t", col.GetName(), col.GetValue(), col.GetUpdated()))
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
