package model

import "time"

type URL struct {
	Hash        string     `gorm:"primaryKey;column:hash"`
	OriginalURL string     `gorm:"column:original_url;not null"`
	UserID      *string    `gorm:"column:user_id"`
	HitCount    int64      `gorm:"column:hit_count;default:0"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
}
