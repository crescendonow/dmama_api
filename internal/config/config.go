package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	CORSOrigins string
	DmamaKey    string
}

func Load() (*Config, error) {
	godotenv.Load()
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Port:        getEnv("PORT", "5013"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),
		DmamaKey:    getEnv("DMAMA_KEY", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
