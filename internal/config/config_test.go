package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set required environment variable
	os.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/test")
	defer os.Unsetenv("MYSQL_DSN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.MySQL.DSN == "" {
		t.Error("MySQL DSN should not be empty")
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("Expected HTTPAddr :8080, got %s", cfg.HTTPAddr)
	}
}

func TestLoad_MissingMySQLDSN(t *testing.T) {
	// Ensure MYSQL_DSN is not set
	os.Unsetenv("MYSQL_DSN")

	_, err := Load()
	if err == nil {
		t.Error("Expected error when MYSQL_DSN is missing")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("MYSQL_DSN", "custom:dsn@tcp(localhost:3306)/custom")
	os.Setenv("REDIS_ADDR", "redis.example.com:6379")
	os.Setenv("REDIS_PASS", "secret")
	os.Setenv("REDIS_DB", "5")
	os.Setenv("HTTP_ADDR", ":9090")

	defer func() {
		os.Unsetenv("MYSQL_DSN")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("REDIS_PASS")
		os.Unsetenv("REDIS_DB")
		os.Unsetenv("HTTP_ADDR")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.MySQL.DSN != "custom:dsn@tcp(localhost:3306)/custom" {
		t.Errorf("Expected custom MySQL DSN, got %s", cfg.MySQL.DSN)
	}

	if cfg.Redis.Addr != "redis.example.com:6379" {
		t.Errorf("Expected custom Redis addr, got %s", cfg.Redis.Addr)
	}

	if cfg.Redis.Password != "secret" {
		t.Errorf("Expected Redis password 'secret', got %s", cfg.Redis.Password)
	}

	if cfg.Redis.DB != 5 {
		t.Errorf("Expected Redis DB 5, got %d", cfg.Redis.DB)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("Expected HTTPAddr :9090, got %s", cfg.HTTPAddr)
	}
}
