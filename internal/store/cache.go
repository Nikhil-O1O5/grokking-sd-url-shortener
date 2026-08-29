package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const urlKeyPrefix = "url:"

type CacheStore struct {
	rdb *redis.Client
}

func NewCacheStore(rdb *redis.Client) *CacheStore {
	return &CacheStore{rdb: rdb}
}

func (c *CacheStore) GetURL(ctx context.Context, hash string) (string, error) {
	val, err := c.rdb.Get(ctx, urlKeyPrefix+hash).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cache get url: %w", err)
	}
	return val, nil
}

func (c *CacheStore) SetURL(ctx context.Context, hash, originalURL string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}

	if err := c.rdb.Set(ctx, urlKeyPrefix+hash, originalURL, ttl).Err(); err != nil {
		return fmt.Errorf("cache set url: %w", err)
	}
	return nil
}

func (c *CacheStore) DeleteURL(ctx context.Context, hash string) error {
	if err := c.rdb.Del(ctx, urlKeyPrefix+hash).Err(); err != nil {
		return fmt.Errorf("cache delete url: %w", err)
	}
	return nil
}
