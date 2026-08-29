package main

import (
	"context"
	"log"
	"sync"

	"github.com/Nikhil-O1O5/url-shortener/internal/store"
	kgspb "github.com/Nikhil-O1O5/url-shortener/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const batchSize = 1000

type kgsServer struct {
	kgspb.UnimplementedKeyGenerationServiceServer
	db   *gorm.DB
	pool []string
	mu   sync.Mutex
}

func newKGSServer(db *gorm.DB) *kgsServer {
	return &kgsServer{db: db}
}

func (s *kgsServer) GetKey(ctx context.Context, req *kgspb.GetKeyRequest) (*kgspb.GetKeyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pool) == 0 {
		keys, err := store.LoadBatch(s.db, batchSize)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load keys: %v", err)
		}
		if len(keys) == 0 {
			return nil, status.Error(codes.ResourceExhausted, "key pool is empty")
		}
		s.pool = keys
		log.Printf("loaded %d keys into memory pool", len(keys))
	}

	key := s.pool[0]
	s.pool = s.pool[1:]

	return &kgspb.GetKeyResponse{Key: key}, nil
}

func (s *kgsServer) ReturnKey(ctx context.Context, req *kgspb.ReturnKeyRequest) (*kgspb.ReturnKeyResponse, error) {
	if err := store.ReturnKey(s.db, req.Key); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to return key: %v", err)
	}
	return &kgspb.ReturnKeyResponse{Success: true}, nil
}
