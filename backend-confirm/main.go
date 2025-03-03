package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Route สำหรับ Root
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to GoGin REST API!"})
	})

	// Route สำหรับดึงข้อมูลทั้งหมด
	//r.GET("/users", getUsers)
	//
	//// Route สำหรับดึงข้อมูลตาม ID
	//r.GET("/users/:id", getUserByID)
	//
	//// Route สำหรับสร้างข้อมูลใหม่
	//r.POST("/users", createUser)
	//
	//// Route สำหรับอัปเดตข้อมูล
	//r.PUT("/users/:id", updateUser)
	//
	//// Route สำหรับลบข้อมูล
	//r.DELETE("/users/:id", deleteUser)

	r.Run(":8082") // เริ่มเซิร์ฟเวอร์ที่พอร์ต 8080
}
