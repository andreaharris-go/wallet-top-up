package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"wallet-top-up/wallet/app/models"
)

func VerifyHandler(db *gorm.DB, producer *kafka.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		type VerifyRequest struct {
			UserID        uint    `json:"user_id" binding:"required"`
			Amount        float64 `json:"amount" binding:"required"`
			PaymentMethod string  `json:"payment_method" binding:"required"`
		}

		var req VerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		if err := db.First(&user, req.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		if req.Amount > 1000.00 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Amount exceeds limit"})
			return
		}

		txID := fmt.Sprintf("tx-%07d", time.Now().Unix())
		expiresAt := time.Now().Add(24 * time.Hour)

		transaction := models.Transaction{
			ID:            txID,
			UserID:        req.UserID,
			Amount:        req.Amount,
			PaymentMethod: req.PaymentMethod,
			Status:        "pending",
			ExpiredAt:     expiresAt,
		}

		if err := db.Create(&transaction).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
			return
		}

		err := producer.WriteMessages(context.Background(), kafka.Message{
			Topic: "transaction_created",
			Key:   []byte(txID),
			Value: []byte("Transaction created"),
		})
		if err != nil {
			fmt.Printf("Failed to publish Kafka message: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish Kafka message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"transaction_id": txID,
			"user_id":        req.UserID,
			"amount":         req.Amount,
			"payment_method": req.PaymentMethod,
			"status":         "pending",
			"expires_at":     expiresAt.Format(time.RFC3339),
		})
	}
}
