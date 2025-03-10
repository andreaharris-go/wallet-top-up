package models

import "time"

type User struct {
	ID      uint    `gorm:"primaryKey"`
	Balance float64 `gorm:"not null;default:0"`
}

type Transaction struct {
	ID                string    `gorm:"primaryKey;type:varchar(50);not null"`
	UserID            uint      `gorm:"not null;index"`
	Amount            float64   `gorm:"not null"`
	PaymentMethod     string    `gorm:"type:varchar(20);not null"`
	Status            string    `gorm:"type:varchar(20);not null"`
	Balance           float64   `gorm:"not null"`
	PrevTransactionID string    `gorm:"type:varchar(50);not null;index"`
	PrevBalance       float64   `gorm:"not null"`
	ExpiredAt         time.Time `gorm:"not null"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}
