package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Structs สำหรับ GORM
type User struct {
	ID      uint    `gorm:"primaryKey"`
	Balance float64 `gorm:"not null;default:0"`
}

type Transaction struct {
	ID                string    `gorm:"primaryKey"`
	UserID            uint      `gorm:"not null;index"`
	Amount            float64   `gorm:"not null"`
	PaymentMethod     string    `gorm:"type:varchar(20);not null"`
	Status            string    `gorm:"type:varchar(20);not null"`
	Balance           float64   `gorm:"not null"`
	PrevTransactionID string    `gorm:"not null;index"`
	PrevBalance       float64   `gorm:"not null"`
	ExpiredAt         time.Time `gorm:"not null"`
}

const (
	host     = "localhost"
	port     = 5432
	user     = "user"
	password = "password"
	dbname   = "mydb"
)

func main() {
	// สร้าง DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// เปิดการเชื่อมต่อกับ PostgreSQL โดยใช้ GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")

	// รีเซ็ตตาราง (ลบและสร้างใหม่)
	err = resetTables(db)
	if err != nil {
		log.Fatal("Error resetting tables:", err)
	}

	// เพิ่มข้อมูล users 10 records
	err = insertUsers(db)
	if err != nil {
		log.Fatal("Error inserting users:", err)
	}

	fmt.Println("Database setup complete!")
}

// ฟังก์ชันรีเซ็ตตาราง
func resetTables(db *gorm.DB) error {
	// ลบตารางทั้งหมดแล้วสร้างใหม่
	err := db.Migrator().DropTable(&Transaction{}, &User{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&User{}, &Transaction{})
	if err != nil {
		return err
	}

	fmt.Println("Tables reset successfully!")
	return nil
}

// ฟังก์ชันเพิ่มข้อมูล users 10 records
func insertUsers(db *gorm.DB) error {
	var users []User
	for i := 1; i <= 10; i++ {
		users = append(users, User{ID: uint(i), Balance: 1000})
	}
	err := db.Create(&users).Error
	if err != nil {
		return err
	}

	fmt.Println("Inserted 10 users successfully!")
	return nil
}
