package store

import (
	"fmt"

	"github.com/Nikhil-O1O5/url-shortener/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KeyStore struct {
	db *gorm.DB
}

func NewKeyStore(db *gorm.DB) *KeyStore {
	return &KeyStore{db: db}
}

func (s *KeyStore) ReturnKey(key string) error {
	return ReturnKey(s.db, key)
}

func (s *KeyStore) BulkReturnKeys(keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.UsedKey{}, "key IN ?", keys).Error; err != nil {
			return fmt.Errorf("delete used keys: %w", err)
		}
		unused := make([]model.UnusedKey, len(keys))
		for i, k := range keys {
			unused[i] = model.UnusedKey{Key: k}
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&unused).Error
	})
}

func LoadBatch(db *gorm.DB, size int) ([]string, error) {
	var keys []model.UnusedKey

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Limit(size).Find(&keys).Error; err != nil {
			return err
		}

		if len(keys) == 0 {
			return nil
		}

		keyStrings := make([]string, len(keys))
		for i, k := range keys {
			keyStrings[i] = k.Key
		}

		if err := tx.Delete(&model.UnusedKey{}, "key IN ?", keyStrings).Error; err != nil {
			return err
		}

		usedKeys := make([]model.UsedKey, len(keys))
		for i, k := range keys {
			usedKeys[i] = model.UsedKey{Key: k.Key}
		}

		return tx.Create(&usedKeys).Error
	})

	if err != nil {
		return nil, fmt.Errorf("load batch: %w", err)
	}

	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = k.Key
	}

	return result, nil
}

func ReturnKey(db *gorm.DB, key string) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.UsedKey{}, "key = ?", key).Error; err != nil {
			return err
		}
		return tx.Create(&model.UnusedKey{Key: key}).Error
	})

	if err != nil {
		return fmt.Errorf("return key: %w", err)
	}

	return nil
}

func UnusedKeyCount(db *gorm.DB) (int64, error) {
	var count int64
	if err := db.Model(&model.UnusedKey{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count unused keys: %w", err)
	}
	return count, nil
}

func BulkInsertKeys(db *gorm.DB, keys []model.UnusedKey) error {
	candidates := make([]string, len(keys))
	for i, k := range keys {
		candidates[i] = k.Key
	}

	var alreadyUsed []string
	if err := db.Model(&model.UsedKey{}).
		Where("key IN ?", candidates).
		Pluck("key", &alreadyUsed).Error; err != nil {
		return fmt.Errorf("check used keys: %w", err)
	}

	if len(alreadyUsed) > 0 {
		usedSet := make(map[string]struct{}, len(alreadyUsed))
		for _, k := range alreadyUsed {
			usedSet[k] = struct{}{}
		}
		filtered := keys[:0]
		for _, k := range keys {
			if _, exists := usedSet[k.Key]; !exists {
				filtered = append(filtered, k)
			}
		}
		keys = filtered
	}

	if len(keys) == 0 {
		return nil
	}

	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(keys, len(keys))
	if result.Error != nil {
		return fmt.Errorf("bulk insert keys: %w", result.Error)
	}
	return nil
}
