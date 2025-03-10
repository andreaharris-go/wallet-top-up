package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"wallet-top-up/wallet/app/models"
)

func ConfirmHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type ConfirmRequest struct {
			TransactionID string `json:"transaction_id" binding:"required"`
		}

		var req ConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var transaction models.Transaction
		if err := db.First(&transaction, "id = ?", req.TransactionID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		transaction.Status = "completed"
		if err := db.Save(&transaction).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
			return
		}

		var user models.User
		if err := db.First(&user, transaction.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.Balance += transaction.Amount
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user balance"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"transaction_id": transaction.ID,
			"user_id":        user.ID,
			"amount":         transaction.Amount,
			"status":         transaction.Status,
			"balance":        user.Balance,
		})
	}
}
