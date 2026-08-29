package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	UserID       string     `gorm:"primaryKey;column:user_id"`
	Name         string     `gorm:"column:name;not null"`
	Email        string     `gorm:"column:email;not null;uniqueIndex"`
	PasswordHash string     `gorm:"column:password_hash;not null"`
	APIKey       *string    `gorm:"column:api_key"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	LastLogin    *time.Time `gorm:"column:last_login"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.UserID = uuid.New().String()
	return nil
}
