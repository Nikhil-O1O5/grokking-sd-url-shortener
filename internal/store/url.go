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

func (s *URLStore) IncrementHitCount(hash string) error {
	if err := s.db.Model(&model.URL{}).Where("hash = ?", hash).UpdateColumn("hit_count", gorm.Expr("hit_count + 1")).Error; err != nil {
		return fmt.Errorf("increment hit count: %w", err)
	}
	return nil
}

func (s *URLStore) GetByUserID(userID string) ([]model.URL, error) {
	var urls []model.URL
	if err := s.db.Where("user_id = ? AND expires_at > NOW()", userID).Order("created_at DESC").Find(&urls).Error; err != nil {
		return nil, fmt.Errorf("get urls by user: %w", err)
	}
	return urls, nil
}

func (s *URLStore) Delete(hash string) error {
	result := s.db.Delete(&model.URL{}, "hash = ?", hash)
	if result.Error != nil {
		return fmt.Errorf("delete url: %w", result.Error)
	}
	return nil
}

// DeleteExpired deletes all expired URLs and returns only the KGS-generated hashes
// (is_custom = false) so they can be recycled back into the key pool.
func (s *URLStore) DeleteExpired() ([]string, error) {
	var kgsHashes []string
	err := s.db.Model(&model.URL{}).
		Where("expires_at < NOW() AND is_custom = false").
		Pluck("hash", &kgsHashes).Error
	if err != nil {
		return nil, fmt.Errorf("fetch expired kgs hashes: %w", err)
	}

	if err := s.db.Delete(&model.URL{}, "expires_at < NOW()").Error; err != nil {
		return nil, fmt.Errorf("delete expired urls: %w", err)
	}
	return kgsHashes, nil
}

