package config

import "os"

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	AppDBName        string
	KeyDBName        string
	RedisAddr        string
	KGSAddr          string
	HTTPPort         string
	GRPCPort         string
	JWTSecret        string
	BaseURL          string
}

func Load() Config {
	return Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", ""),
		AppDBName:        getEnv("APP_DB_NAME", "urlshortener"),
		KeyDBName:        getEnv("KEY_DB_NAME", "keydb"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		KGSAddr:          getEnv("KGS_ADDR", "localhost:50051"),
		HTTPPort:         getEnv("HTTP_PORT", ":8080"),
		GRPCPort:         getEnv("GRPC_PORT", ":50051"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		BaseURL:          getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
