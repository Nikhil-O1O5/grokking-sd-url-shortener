package model

import "time"

type UnusedKey struct {
	Key       string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type UsedKey struct {
	Key    string    `gorm:"primaryKey"`
	UsedAt time.Time `gorm:"autoCreateTime"`
}
