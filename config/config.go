package config

import (
	"os"
)

type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	StripeSecretKey string
	StripePublicKey string
	Environment     string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "5000"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:mysecretpassword@localhost:5432/premium_chat?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key"),
		StripeSecretKey: getEnv("STRIPE_SECRET_KEY", "sk_test_..."),
		StripePublicKey: getEnv("STRIPE_PUBLIC_KEY", "pk_test_..."),
		Environment:     getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
