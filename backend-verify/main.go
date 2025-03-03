package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Transaction struct {
	ID            string    `gorm:"primaryKey" json:"transaction_id"`
	UserID        uint      `json:"user_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	ExpiredAt     time.Time `json:"expired_at"`
}

type TransactionMessage struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

var db *gorm.DB
var kafkaProducer *kafka.Producer

const KafkaBroker = "localhost:9092"
const Topic = "transaction"

func initDB() {
	dsn := "host=localhost user=user password=password dbname=mydb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&User{}, &Transaction{})
}

func initKafka() {
	var err error
	kafkaProducer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": KafkaBroker})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
}

func getWallet(c *gin.Context) {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func verifyWallet(c *gin.Context) {
	var req struct {
		UserID        uint    `json:"user_id"`
		Amount        float64 `json:"amount"`
		PaymentMethod string  `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transactionID := uuid.New().String()
	expiredAt := time.Now().Add(5 * time.Minute)

	transaction := Transaction{
		ID:            transactionID,
		UserID:        req.UserID,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		Status:        "pending",
		ExpiredAt:     expiredAt,
	}

	//if err := db.Create(&transaction).Error; err != nil {
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	//	return
	//}

	topic := Topic
	message := TransactionMessage{
		TransactionID: transactionID,
		Status:        transaction.Status,
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	err = kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          msgBytes,
	}, nil)

	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	} else {
		log.Printf("Published: %s", string(msgBytes))
	}

	c.JSON(http.StatusOK, transaction)
}

func main() {
	initDB()
	initKafka()

	r := gin.Default()
	r.GET("/wallet", getWallet)
	r.POST("/wallet/verify", verifyWallet)

	r.Run(":8084")
}
