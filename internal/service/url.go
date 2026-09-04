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
	keyStore   *store.KeyStore
	kgsClient  *kgs.Client
	baseURL    string
}

func NewURLService(urlStore *store.URLStore, cacheStore *store.CacheStore, keyStore *store.KeyStore, kgsClient *kgs.Client, baseURL string) *URLService {
	return &URLService{
		urlStore:   urlStore,
		cacheStore: cacheStore,
		keyStore:   keyStore,
		kgsClient:  kgsClient,
		baseURL:    baseURL,
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
		hash = userAliasHash(req.UserID, req.CustomAlias)
		existing, err := s.urlStore.GetByHash(hash)
		if err != nil {
			return nil, fmt.Errorf("check alias: %w", err)
		}
		if existing != nil {
			return nil, ErrCustomAliasTaken
		}
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
		IsCustom:    req.CustomAlias != "",
		ExpiresAt:   expiresAt,
	}

	if err := s.urlStore.Create(url); err != nil {
		return nil, fmt.Errorf("store url: %w", err)
	}

	return &ShortenResponse{
		Hash:      hash,
		ShortURL:  fmt.Sprintf("%s/%s", s.baseURL, hash),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *URLService) ResolveURL(ctx context.Context, hash string) (*model.URL, error) {
	if originalURL, err := s.cacheStore.GetURL(ctx, hash); err == nil && originalURL != "" {
		go s.urlStore.IncrementHitCount(hash)
		return &model.URL{Hash: hash, OriginalURL: originalURL}, nil
	}

	url, err := s.urlStore.GetByHash(hash)
	if err != nil {
		return nil, fmt.Errorf("resolve url: %w", err)
	}
	if url == nil {
		return nil, nil
	}

	if time.Now().After(url.ExpiresAt) {
		_ = s.urlStore.Delete(hash)
		_ = s.cacheStore.DeleteURL(ctx, hash)
		if !url.IsCustom {
			_ = s.keyStore.ReturnKey(hash)
		}
		return nil, nil
	}

	go s.urlStore.IncrementHitCount(hash)
	_ = s.cacheStore.SetURL(ctx, hash, url.OriginalURL, url.ExpiresAt)
	return url, nil
}

type StatsResponse struct {
	Hash        string    `json:"hash"`
	HitCount    int64     `json:"hit_count"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	OriginalURL *string   `json:"original_url,omitempty"`
}

func (s *URLService) GetStats(ctx context.Context, hash string, requesterUserID *string) (*StatsResponse, error) {
	url, err := s.urlStore.GetByHash(hash)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	if url == nil {
		return nil, nil
	}

	resp := &StatsResponse{
		Hash:      url.Hash,
		HitCount:  url.HitCount,
		CreatedAt: url.CreatedAt,
		ExpiresAt: url.ExpiresAt,
	}

	isOwner := requesterUserID != nil && url.UserID != nil && *requesterUserID == *url.UserID
	if isOwner {
		resp.OriginalURL = &url.OriginalURL
	}

	return resp, nil
}

func (s *URLService) GetUserURLs(ctx context.Context, userID string) ([]model.URL, error) {
	urls, err := s.urlStore.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("get user urls: %w", err)
	}
	return urls, nil
}

// userAliasHash scopes a custom alias to its owner so two users can use the same alias
// without conflicting. Uses the first 8 chars of the user UUID as a prefix.
// e.g. user 550e8400-... with alias "mylink" → "550e8400-mylink"
func userAliasHash(userID *string, alias string) string {
	if userID == nil {
		return alias
	}
	prefix := *userID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	return prefix + "-" + alias
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
