package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID      uint    `gorm:"primaryKey" json:"user_id"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID                string    `gorm:"primaryKey" json:"transaction_id"`
	UserID            uint      `json:"user_id"`
	Amount            float64   `json:"amount"`
	PaymentMethod     string    `json:"payment_method"`
	Status            string    `json:"status"`
	Balance           float64   `json:"balance"`
	PrevTransactionID string    `json:"prev_transaction_id"`
	PrevBalance       float64   `json:"prev_balance"`
	ExpiredAt         time.Time `json:"expired_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type TransactionMessage struct {
	TransactionID string    `json:"transaction_id"`
	UserID        uint      `json:"user_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	ExpiredAt     time.Time `json:"expired_at"`
}

var db *gorm.DB
var kafkaProducer *kafka.Producer

const (
	KafkaBroker    = "localhost:9092"
	Topic          = "transaction"
	GraylogAddress = "http://graylog:12201/gelf"
)

func initDB() {
	dsn := "host=localhost user=user password=password dbname=mydb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}

func initKafka() {
	var err error
	kafkaProducer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": KafkaBroker})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
}

func sendLogToGraylog(level int8, message string, extra map[string]interface{}) {
	logData := map[string]interface{}{
		"host":          "localhost",
		"short_message": message,
		"timestamp":     time.Now().Unix(),
		"level":         level,
		"_custom_field": "production",
		"version":       "1.1",
	}

	for k, v := range extra {
		logData[k] = v
	}

	jsonData, err := json.Marshal(logData)
	if err != nil {
		log.Printf("Failed to marshal log data: %v", err)
		return
	}

	req, err := http.NewRequest("POST", GraylogAddress, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send log to Graylog: %v", err)
	} else {
		log.Printf("send log")
		defer resp.Body.Close()
	}

	log.Printf("End log")
}

func getWallet(c *gin.Context) {
	var users []User
	if err := db.Order("id").Find(&users).Error; err != nil {
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

	var user User
	if err := db.First(&user, req.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendLogToGraylog(3, "USER_NOT_FOUND", map[string]interface{}{"error": "User not found"})
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	transactionID := fmt.Sprintf("txn-%d%d", time.Now().Unix(), rand.Intn(10000))
	expiredAt := time.Now().Add(5 * time.Minute)
	createdAt := time.Now()

	transaction := Transaction{
		ID:                transactionID,
		UserID:            req.UserID,
		Amount:            req.Amount,
		PaymentMethod:     req.PaymentMethod,
		Status:            "pending",
		Balance:           0,
		PrevTransactionID: "none",
		PrevBalance:       0,
		ExpiredAt:         expiredAt,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	}

	if err := db.Create(&transaction).Error; err != nil {
		sendLogToGraylog(3, "CREATE_TRANSACTION_ERROR", map[string]interface{}{"error": "Create error"})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publishMessage(transaction)

	c.JSON(http.StatusOK, transaction)
}

func confirmWallet(c *gin.Context) {
	var req struct {
		TransactionID string `json:"transaction_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var transaction Transaction
	if err := db.Where("id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sendLogToGraylog(3, "TRANSACTION_NOT_FOUND", map[string]interface{}{"error": "Transaction not found"})
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	if transaction.Status != "completed" {
		if transaction.Status == "verified" {
			result := db.Model(&Transaction{}).Where("id = ?", transaction.ID).Updates(map[string]interface{}{
				"status":     "confirming",
				"updated_at": time.Now(),
			})

			if result.Error != nil {
				log.Printf("Failed to update transaction status: %v", result.Error)
			} else {
				transaction.Status = "confirming"
				publishMessage(transaction)
			}
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"transaction_id": transaction.ID,
			"message":        "Transaction is not completed",
		})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func publishMessage(transaction Transaction) {
	topic := Topic
	message := TransactionMessage{
		TransactionID: transaction.ID,
		UserID:        transaction.UserID,
		Amount:        transaction.Amount,
		PaymentMethod: transaction.PaymentMethod,
		Status:        transaction.Status,
		ExpiredAt:     transaction.ExpiredAt,
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
}

func main() {
	initDB()
	initKafka()

	r := gin.Default()
	r.GET("/wallet", getWallet)
	r.POST("/wallet/verify", verifyWallet)
	r.POST("/wallet/confirm", confirmWallet)

	err := r.Run(":8084")
	if err != nil {
		log.Printf("Server start error: %v", err)
		return
	}
}
