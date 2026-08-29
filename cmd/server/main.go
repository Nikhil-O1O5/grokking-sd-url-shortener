package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Nikhil-O1O5/url-shortener/internal/config"
	"github.com/Nikhil-O1O5/url-shortener/internal/handler"
	"github.com/Nikhil-O1O5/url-shortener/internal/kgs"
	"github.com/Nikhil-O1O5/url-shortener/internal/service"
	"github.com/Nikhil-O1O5/url-shortener/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

	appDB, err := store.NewPostgresDB(store.PostgresConfig{
		Host:     cfg.PostgresHost,
		Port:     cfg.PostgresPort,
		User:     cfg.PostgresUser,
		Password: cfg.PostgresPassword,
		DBName:   cfg.AppDBName,
	})
	if err != nil {
		log.Fatalf("failed to connect to urlshortener db: %v", err)
	}
	log.Println("connected to urlshortener db")

	_, err = store.NewRedisClient(store.RedisConfig{
		Addr: cfg.RedisAddr,
	})
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("connected to redis")

	kgsClient, err := kgs.NewClient(cfg.KGSAddr)
	if err != nil {
		log.Fatalf("failed to connect to KGS: %v", err)
	}
	defer kgsClient.Close()
	log.Println("connected to KGS")

	urlStore  := store.NewURLStore(appDB)
	userStore := store.NewUserStore(appDB)

	urlService  := service.NewURLService(urlStore, kgsClient)
	authService := service.NewAuthService(userStore, cfg.JWTSecret)

	urlHandler  := handler.NewURLHandler(urlService, authService)
	authHandler := handler.NewAuthHandler(authService)

	r := handler.NewRouter(urlHandler, authHandler)

	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("app server listening on %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down app server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("app server stopped")
}
