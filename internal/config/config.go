package config

import (
	"os"
	"strconv"
)

type Config struct {
	DB_URL                string
	MAX_UPLOAD_SIZE_BYTES uint64
	STORAGE_NODE_URLS     string
	STORAGE_NODE_TOKEN    string
}

func LoadConfig() *Config {
	cfg := &Config{
		DB_URL:                getEnv("DB_URL", "postgres://storage:storage@localhost:5432/storage"),
		MAX_UPLOAD_SIZE_BYTES: getEnvAsUint64("MAX_UPLOAD_SIZE_MB", 100), // default 100MB,
		STORAGE_NODE_URLS:     getEnv("STORAGE_NODE_URLS", "http://localhost:8081"),
		STORAGE_NODE_TOKEN:    getEnv("STORAGE_NODE_TOKEN", ""),
	}
	cfg.MAX_UPLOAD_SIZE_BYTES = cfg.MAX_UPLOAD_SIZE_BYTES << 20
	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsUint64(key string, defaultValue uint64) uint64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
