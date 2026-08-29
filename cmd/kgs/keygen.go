package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Nikhil-O1O5/url-shortener/internal/model"
	"github.com/Nikhil-O1O5/url-shortener/internal/store"
	"gorm.io/gorm"
)

const (
	charset       = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	keyLength     = 6
	seedThreshold = 10_000
	seedTarget    = 100_000
	insertBatch   = 500
)

func generateKey() (string, error) {
	key := make([]byte, keyLength)
	for i := range key {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		key[i] = charset[n.Int64()]
	}
	return string(key), nil
}

func seedKeysIfNeeded(db *gorm.DB) error {
	count, err := store.UnusedKeyCount(db)
	if err != nil {
		return err
	}

	if count >= seedThreshold {
		fmt.Printf("unused key pool has %d keys, no seeding needed\n", count)
		return nil
	}

	needed := seedTarget - int(count)
	fmt.Printf("unused key pool has %d keys, generating %d more...\n", count, needed)

	batch := make([]model.UnusedKey, 0, insertBatch)
	generated := 0

	for generated < needed {
		key, err := generateKey()
		if err != nil {
			return fmt.Errorf("generate key: %w", err)
		}

		batch = append(batch, model.UnusedKey{Key: key})
		generated++

		if len(batch) == insertBatch {
			if err := store.BulkInsertKeys(db, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := store.BulkInsertKeys(db, batch); err != nil {
			return err
		}
	}

	fmt.Printf("seeded %d keys into unused_keys\n", needed)
	return nil
}

func startTopUpWorker(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := seedKeysIfNeeded(db); err != nil {
				log.Printf("top-up worker: %v", err)
			}
		case <-ctx.Done():
			log.Println("top-up worker: stopped")
			return
		}
	}
}
