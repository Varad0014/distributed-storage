package config

import (
	"os"
	"strconv"
)

type Config struct {
	DB_URL string
	MAX_UPLOAD_SIZE_BYTES uint64
	STORAGE_NODE_URL string
	STORAGE_NODE_URL_2 string
}

func LoadConfig() *Config{
	cfg := &Config{
		DB_URL: getEnv("DB_URL", "postgres://storage:storage@localhost:5432/storage"),
		MAX_UPLOAD_SIZE_BYTES: getEnvAsUint64("MAX_UPLOAD_SIZE_MB", 100), // default 100MB,
		STORAGE_NODE_URL: getEnv("STORAGE_NODE_URL", "http://localhost:8081"),
		STORAGE_NODE_URL_2: getEnv("STORAGE_NODE_URL_2", "http://localhost:8082"),
	}
	cfg.MAX_UPLOAD_SIZE_BYTES = cfg.MAX_UPLOAD_SIZE_BYTES << 20
	return cfg
}

func getEnv(key, defaultValue string) string{
	value := os.Getenv(key)
	if value == ""{
		return defaultValue
	}
	return value
}

func getEnvAsUint64(key string, defaultValue uint64) uint64{
	valueStr := os.Getenv(key)
	if valueStr == ""{
		return defaultValue
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil{
		return defaultValue
	}
	return value
}

