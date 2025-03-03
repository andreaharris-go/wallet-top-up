package main

import (
	"context"
	"encoding/json"
	_ "fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Transaction struct {
	ID                string    `gorm:"primaryKey" json:"transaction_id"`
	UserID            uint      `json:"user_id"`
	Amount            float64   `json:"amount"`
	PaymentMethod     string    `json:"payment_method"`
	Status            string    `json:"status"`
	Balance           int       `json:"balance"`
	PrevTransactionID string    `json:"prev_transaction_id"`
	PrevBalance       int       `json:"prev_balance"`
	ExpiredAt         time.Time `json:"expired_at"`
}

type TransactionMessage struct {
	TransactionID string    `json:"transaction_id"`
	UserID        uint      `json:"user_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	ExpiredAt     time.Time `json:"expired_at"`
}

var (
	ctx         = context.Background()
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	db *gorm.DB
)

const KafkaBroker = "localhost:9092"
const Topic = "transaction"

func initDB() {
	var err error
	dsn := "host=localhost user=user password=password dbname=mydb port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}

func consumeKafka() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": KafkaBroker,
		"group.id":          "transaction-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer func(consumer *kafka.Consumer) {
		err := consumer.Close()
		if err != nil {
			log.Printf("consumer close error: %v", err)
		}
	}(consumer)

	err = consumer.Subscribe("transaction", nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to Kafka topic: %v", err)
	}

	for {
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			var txn TransactionMessage
			if err := json.Unmarshal(msg.Value, &txn); err != nil {
				log.Printf("Failed to parse Kafka message: %v", err)
				continue
			}

			exists, err := redisClient.Exists(ctx, txn.TransactionID).Result()
			if err != nil {
				log.Printf("Redis error: %v", err)
				continue
			}
			if exists > 0 {
				log.Printf("Transaction %s already processed", txn.TransactionID)
				continue
			}

			redisClient.Set(ctx, txn.TransactionID, "processed", 10*time.Minute)

			result := db.Model(&Transaction{}).Where("id = ?", txn.TransactionID).Update("status", "completed")
			if result.Error != nil {
				log.Printf("Failed to update transaction status: %v", result.Error)
				continue
			}

			publishKafka(txn.TransactionID)
		} else {
			log.Printf("Consumer error: %v", err)
		}
	}
}

func publishKafka(transactionID string) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": KafkaBroker})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	msg := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         "completed",
	}
	value, _ := json.Marshal(msg)

	topic := Topic
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, nil)

	if err != nil {
		return
	}

	producer.Flush(500)
}

func main() {
	initDB()
	consumeKafka()
}
