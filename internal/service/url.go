package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nikhil-O1O5/url-shortener/internal/kgs"
	"github.com/Nikhil-O1O5/url-shortener/internal/model"
	"github.com/Nikhil-O1O5/url-shortener/internal/store"
)

const (
	defaultAnonExpiry = 30 * 24 * time.Hour      // 30 days
	defaultUserExpiry = 2 * 365 * 24 * time.Hour // 2 years
)

var ErrCustomAliasTaken = errors.New("custom alias already taken")

type URLService struct {
	urlStore   *store.URLStore
	cacheStore *store.CacheStore
	kgsClient  *kgs.Client
}

func NewURLService(urlStore *store.URLStore, cacheStore *store.CacheStore, kgsClient *kgs.Client) *URLService {
	return &URLService{
		urlStore:   urlStore,
		cacheStore: cacheStore,
		kgsClient:  kgsClient,
	}
}

type ShortenRequest struct {
	OriginalURL string
	CustomAlias string
	UserID      *string
	ExpireAt    *time.Time
}

type ShortenResponse struct {
	Hash      string
	ShortURL  string
	ExpiresAt time.Time
}

func (s *URLService) ShortenURL(ctx context.Context, req ShortenRequest) (*ShortenResponse, error) {
	var hash string

	if req.CustomAlias != "" {
		existing, err := s.urlStore.GetByHash(req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("check alias: %w", err)
		}
		if existing != nil {
			return nil, ErrCustomAliasTaken
		}
		hash = req.CustomAlias
	} else {
		key, err := s.kgsClient.GetKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("get key: %w", err)
		}
		hash = key
	}

	expiresAt := defaultExpiry(req.UserID, req.ExpireAt)

	url := &model.URL{
		Hash:        hash,
		OriginalURL: req.OriginalURL,
		UserID:      req.UserID,
		ExpiresAt:   expiresAt,
	}

	if err := s.urlStore.Create(url); err != nil {
		return nil, fmt.Errorf("store url: %w", err)
	}

	return &ShortenResponse{
		Hash:      hash,
		ShortURL:  fmt.Sprintf("http://localhost:8080/%s", hash),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *URLService) ResolveURL(ctx context.Context, hash string) (*model.URL, error) {
	if originalURL, err := s.cacheStore.GetURL(ctx, hash); err == nil && originalURL != "" {
		return &model.URL{Hash: hash, OriginalURL: originalURL}, nil
	}

	url, err := s.urlStore.GetByHash(hash)
	if err != nil {
		return nil, fmt.Errorf("resolve url: %w", err)
	}
	if url == nil {
		return nil, nil
	}

	_ = s.cacheStore.SetURL(ctx, hash, url.OriginalURL, url.ExpiresAt)
	return url, nil
}

func defaultExpiry(userID *string, custom *time.Time) time.Time {
	if custom != nil {
		return *custom
	}
	if userID != nil {
		return time.Now().Add(defaultUserExpiry)
	}
	return time.Now().Add(defaultAnonExpiry)
}
