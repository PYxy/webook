package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/go-redis/redis"
)

func main() {
	// Create a new Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "192.168.188.199:6379",
		Password: "", // If your Redis instance has authentication enabled
		DB:       0,  // Select the appropriate database
	})

	// Create a new Kafka producer
	//kafkaProducer, err := sarama.NewSyncProducer([]string{"115.238.186.133:59092"}, nil)
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Create a channel to receive OS interrupt signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Subscribe to Redis Pub/Sub channel
	pubsub := redisClient.Subscribe("suricata")
	defer pubsub.Close()

	// Retrieve the subscription channel
	channel := pubsub.Channel()
	fmt.Println(len(channel))
	// Start listening for new Redis messages
	go func() {
		for msg := range channel {
			value := []byte(msg.Payload)

			//// Send the message to Kafka
			//kafkaProducer.SendMessage(&sarama.ProducerMessage{
			//	Topic: "test",
			//	Value: sarama.ByteEncoder(value),
			//})

			fmt.Printf("Sent message to Kafka: %s\n", value)
		}
	}()

	// Wait for OS interrupt signal to exit
	<-signals
}
