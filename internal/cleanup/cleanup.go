package cleanup

import (
	"context"
	"log"
	"time"

	"github.com/Nikhil-O1O5/url-shortener/internal/store"
)

const sweepInterval = 1 * time.Hour

type Worker struct {
	urlStore   *store.URLStore
	cacheStore *store.CacheStore
	keyStore   *store.KeyStore
}

func NewWorker(urlStore *store.URLStore, cacheStore *store.CacheStore, keyStore *store.KeyStore) *Worker {
	return &Worker{urlStore: urlStore, cacheStore: cacheStore, keyStore: keyStore}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.sweep(ctx)
		case <-ctx.Done():
			log.Println("cleanup worker stopped")
			return
		}
	}
}

func (w *Worker) sweep(ctx context.Context) {
	hashes, err := w.urlStore.DeleteExpired()
	if err != nil {
		log.Printf("cleanup sweep error: %v", err)
		return
	}
	if len(hashes) == 0 {
		return
	}

	// Evict from Redis cache
	for _, hash := range hashes {
		_ = w.cacheStore.DeleteURL(ctx, hash)
	}

	if err := w.keyStore.BulkReturnKeys(hashes); err != nil {
		log.Printf("cleanup: failed to return keys to pool: %v", err)
	}

	log.Printf("cleanup sweep: deleted %d expired URLs", len(hashes))
}
