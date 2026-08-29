package store

import (
	"errors"
	"fmt"

	"github.com/Nikhil-O1O5/url-shortener/internal/model"
	"gorm.io/gorm"
)

type URLStore struct {
	db *gorm.DB
}

func NewURLStore(db *gorm.DB) *URLStore {
	return &URLStore{db: db}
}

func (s *URLStore) Create(url *model.URL) error {
	result := s.db.Create(url)
	if result.Error != nil {
		return fmt.Errorf("create url: %w", result.Error)
	}
	return nil
}

func (s *URLStore) GetByHash(hash string) (*model.URL, error) {
	var url model.URL
	result := s.db.Where("hash = ?", hash).First(&url)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("get url by hash: %w", result.Error)
	}
	return &url, nil
}

