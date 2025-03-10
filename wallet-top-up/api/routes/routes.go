package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"wallet-top-up/api/handlers"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, producer *kafka.Writer) {
	router.POST("/wallet/verify", handlers.VerifyHandler(db, producer))
	router.POST("/wallet/confirm", handlers.ConfirmHandler(db))
}
