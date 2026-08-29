package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nikhil-O1O5/url-shortener/internal/config"
	"github.com/Nikhil-O1O5/url-shortener/internal/store"
	kgspb "github.com/Nikhil-O1O5/url-shortener/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

	db, err := store.NewPostgresDB(store.PostgresConfig{
		Host:     cfg.PostgresHost,
		Port:     cfg.PostgresPort,
		User:     cfg.PostgresUser,
		Password: cfg.PostgresPassword,
		DBName:   cfg.KeyDBName,
	})
	if err != nil {
		log.Fatalf("failed to connect to keydb: %v", err)
	}
	log.Println("connected to keydb")

	if err := seedKeysIfNeeded(db); err != nil {
		log.Fatalf("failed to seed keys: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startTopUpWorker(ctx, db)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	kgspb.RegisterKeyGenerationServiceServer(grpcServer, newKGSServer(db))
	log.Printf("KGS gRPC server listening on %s", cfg.GRPCPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down KGS...")
	cancel()
	grpcServer.GracefulStop()
	log.Println("KGS stopped")
}
