package test_demo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func TestRedisCus(t *testing.T) {
	type Mst struct {
		Age  int    `json:"age"`
		Name string `json:"name"`
	}
	//a := Mst{
	//	Age:  18,
	//	Name: "小臂",
	//}

	rdb := redis.NewClient(&redis.Options{
		Addr: "120.132.118.90:58247",
	})
	c1, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if rdb.Ping(c1).Err() != nil {
		panic("redis连接失败")
	}
	channel := "my_channel"
	go func() {
		for {
			pub := rdb.Publish(ctx, channel, "id:70").Err()
			fmt.Println("数据发布失败:", pub)
			time.Sleep(time.Second * 2)
		}

	}()

	//pubsub := rdb.Subscribe(ctx, "channel1", "channel2", "channel3") // 订阅多个频道
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()

	// 接收消息
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(msg.Channel, msg.Payload)
	}
}
