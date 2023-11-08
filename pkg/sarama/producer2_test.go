package sarama

import (
	"log"
	"os"
	"os/signal"
	"testing"

	"github.com/IBM/sarama"
)

func Test_KafkaConsumer(t *testing.T) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	client, err := sarama.NewClient([]string{"120.132.118.90:9094"}, config)
	defer client.Close()
	if err != nil {
		panic(err)
	}
	consumer, err := sarama.NewConsumerFromClient(client)

	defer consumer.Close()
	if err != nil {
		panic(err)
	}
	// get partitionId list
	partitions, err := consumer.Partitions("ljy")
	if err != nil {
		panic(err)
	}

	for _, partitionId := range partitions {
		// create partitionConsumer for every partitionId
		partitionConsumer, err := consumer.ConsumePartition("120.132.118.90:9094", partitionId, sarama.OffsetOldest)
		if err != nil {
			panic(err)
		}

		go func(pc *sarama.PartitionConsumer) {
			defer (*pc).Close()
			// block
			for message := range (*pc).Messages() {
				value := string(message.Value)
				log.Printf("Partitionid: %d; offset:%d, value: %s\n", message.Partition, message.Offset, value)
			}

		}(&partitionConsumer)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	select {
	case <-signals:

	}
}
