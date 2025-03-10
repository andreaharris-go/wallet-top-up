package utils

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
)

func NewKafkaProducer(brokerAddress string) (*kafka.Writer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Balancer: &kafka.LeastBytes{},
	}

	log.Println("Kafka producer initialized")
	return writer, nil
}

func PublishMessage(writer *kafka.Writer, topic string, key, value []byte) error {
	err := writer.WriteMessages(
		context.Background(),
		kafka.Message{
			Topic: topic,
			Key:   key,
			Value: value,
		},
	)
	
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	log.Printf("Message published to topic %s", topic)
	return nil
}
