package models

import "gorm.io/gorm"

type Article struct {
	gorm.Model
	Title   string `bingding:"required" `
	Content string `bingding:"required" `
	Preview string `bingding:"required" `
	// Likes   int    `gorm:"default:0"`

	// UserID  uint   `json:"user_id"` // Foreign key to User model
	// User    User   `gorm:"foreignKey:UserID"` // Relationship to User model
}
