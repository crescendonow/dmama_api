package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	GISDataURL   string
	Port         string
	CORSOrigins  string
	DmamaKey     string
	MongoURI     string
	MongoDB      string
	MongoUsersDB string
	SystemUser   string
	UserDomain   string
}

func Load() (*Config, error) {
	godotenv.Load()
	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		GISDataURL:   getEnv("GISDATA_URL", ""),
		Port:         getEnv("PORT", "5013"),
		CORSOrigins:  getEnv("CORS_ORIGINS", "*"),
		DmamaKey:     getEnv("DMAMA_KEY", ""),
		MongoURI:     getEnv("MONGO_URI", ""),
		MongoDB:      getEnv("MONGO_DB", "vallaris_feature"),
		MongoUsersDB: getEnv("MONGO_USERS_DB", "claystone"),
		SystemUser:   getEnv("SYSTEM_USER_ID", ""),
		UserDomain:   getEnv("USER_DOMAIN", "pwa.co.th"),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
