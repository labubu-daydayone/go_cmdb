package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration
type Config struct {
	MySQL    MySQLConfig
	Redis    RedisConfig
	HTTPAddr string
}

// MySQLConfig holds MySQL configuration
type MySQLConfig struct {
	DSN string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		MySQL: MySQLConfig{
			DSN: getEnv("MYSQL_DSN", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASS", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
	}

	// Validate required fields
	if cfg.MySQL.DSN == "" {
		return nil, fmt.Errorf("MYSQL_DSN is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
