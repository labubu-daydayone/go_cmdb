package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/ini.v1"
)

// Config holds all configuration
type Config struct {
	MySQL            MySQLConfig
	Redis            RedisConfig
	JWT              JWTConfig
	Migrate          bool
	HTTPAddr         string
	AgentToken       string // Deprecated: use mTLS instead
	MTLS             MTLSConfig
	RiskScanner      RiskScannerConfig
	ReleaseExecutor  ReleaseExecutorConfig
	DNSWorker        DNSWorkerConfig
	ACMEWorker       ACMEWorkerConfig
	CertCleaner      CertCleanerConfig
	NodeHealthWorker NodeHealthWorkerConfig
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

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret        string
	ExpireMinutes int
	Issuer        string
}

// MTLSConfig holds mTLS configuration
type MTLSConfig struct {
	Enabled    bool
	ClientCert string
	ClientKey  string
	CACert     string
}

// RiskScannerConfig holds risk scanner configuration
type RiskScannerConfig struct {
	Enabled               bool
	IntervalSec           int
	CertExpiringDays      int
	CertExpiringThreshold int
	ACMEMaxAttempts       int
}

// ReleaseExecutorConfig holds release executor configuration
type ReleaseExecutorConfig struct {
	Enabled     bool
	IntervalSec int
}

// DNSWorkerConfig holds DNS worker configuration
type DNSWorkerConfig struct {
	Enabled     bool
	IntervalSec int
	BatchSize   int
}

// ACMEWorkerConfig holds ACME worker configuration
type ACMEWorkerConfig struct {
	Enabled     bool
	IntervalSec int
	BatchSize   int
}

// CertCleanerConfig holds certificate cleaner configuration
type CertCleanerConfig struct {
	Enabled        bool
	IntervalSec    int
	FailedKeepDays int
}

// NodeHealthWorkerConfig holds node health worker configuration
type NodeHealthWorkerConfig struct {
	Enabled              bool
	IntervalSec          int
	TimeoutSec           int
	Concurrency          int
	OfflineFailThreshold int
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
		JWT: JWTConfig{
			Secret:        os.Getenv("JWT_SECRET"),
			ExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 1440),
			Issuer:        getEnv("JWT_ISSUER", "go_cmdb"),
		},
		Migrate:    getEnv("MIGRATE", "0") == "1",
		HTTPAddr:   getEnv("HTTP_ADDR", ":8080"),
		AgentToken: getEnv("AGENT_TOKEN", ""), // Deprecated
			MTLS: MTLSConfig{
				Enabled:    getEnv("MTLS_ENABLED", "0") == "1",
				ClientCert: getEnv("CONTROL_CERT", ""),
				ClientKey:  getEnv("CONTROL_KEY", ""),
				CACert:     getEnv("CONTROL_CA", ""),
			},
			RiskScanner: RiskScannerConfig{
				Enabled:               getEnv("RISK_SCANNER_ENABLED", "1") == "1",
				IntervalSec:           getEnvInt("RISK_SCANNER_INTERVAL_SEC", 300),
				CertExpiringDays:      getEnvInt("CERT_EXPIRING_DAYS", 15),
				CertExpiringThreshold: getEnvInt("CERT_EXPIRING_WEBSITE_THRESHOLD", 2),
				ACMEMaxAttempts:       getEnvInt("ACME_MAX_ATTEMPTS", 3),
			},
		ReleaseExecutor: ReleaseExecutorConfig{
			Enabled:     getEnv("RELEASE_EXECUTOR_ENABLED", "1") == "1",
			IntervalSec: getEnvInt("RELEASE_EXECUTOR_INTERVAL_SEC", 5),
		},
		DNSWorker: DNSWorkerConfig{
			Enabled:     getEnv("DNS_WORKER_ENABLED", "1") == "1",
			IntervalSec: getEnvInt("DNS_WORKER_INTERVAL_SEC", 30),
			BatchSize:   getEnvInt("DNS_WORKER_BATCH_SIZE", 10),
		},
	}

	// Validate required fields
	if cfg.MySQL.DSN == "" {
		return nil, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
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

// LoadFromINI loads configuration from INI file with environment variable override
func LoadFromINI(iniPath string) (*Config, error) {
	// Load INI file
	cfgFile, err := ini.Load(iniPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load INI file: %w", err)
	}

	// Helper function: get value with priority: ENV > INI > default
	getValue := func(envKey, iniSection, iniKey, defaultValue string) string {
		// Priority 1: Environment variable
		if value := os.Getenv(envKey); value != "" {
			return value
		}
		// Priority 2: INI file
		if value := cfgFile.Section(iniSection).Key(iniKey).String(); value != "" {
			return value
		}
		// Priority 3: Default value
		return defaultValue
	}

	getValueInt := func(envKey, iniSection, iniKey string, defaultValue int) int {
		// Priority 1: Environment variable
		if value := os.Getenv(envKey); value != "" {
			if intValue, err := strconv.Atoi(value); err == nil {
				return intValue
			}
		}
		// Priority 2: INI file
		if cfgFile.Section(iniSection).HasKey(iniKey) {
			if value, err := cfgFile.Section(iniSection).Key(iniKey).Int(); err == nil {
				return value
			}
		}
		// Priority 3: Default value
		return defaultValue
	}

	getValueBool := func(envKey, iniSection, iniKey string, defaultValue bool) bool {
		// Priority 1: Environment variable
		if value := os.Getenv(envKey); value != "" {
			return value == "1" || value == "true"
		}
		// Priority 2: INI file
		if value, err := cfgFile.Section(iniSection).Key(iniKey).Bool(); err == nil {
			return value
		}
		// Priority 3: Default value
		return defaultValue
	}

	cfg := &Config{
		MySQL: MySQLConfig{
			DSN: getValue("MYSQL_DSN", "mysql", "dsn", ""),
		},
		Redis: RedisConfig{
			Addr:     getValue("REDIS_ADDR", "redis", "addr", "localhost:6379"),
			Password: getValue("REDIS_PASS", "redis", "pass", ""),
			DB:       getValueInt("REDIS_DB", "redis", "db", 0),
		},
		JWT: JWTConfig{
			Secret:        getValue("JWT_SECRET", "jwt", "secret", ""),
			ExpireMinutes: getValueInt("JWT_EXPIRE_MINUTES", "jwt", "expire_seconds", 86400) / 60,
			Issuer:        getValue("JWT_ISSUER", "jwt", "issuer", "go_cmdb"),
		},
		Migrate:    getValueBool("MIGRATE", "app", "migrate", false),
		HTTPAddr:   getValue("HTTP_ADDR", "http", "addr", ":8080"),
		AgentToken: getValue("AGENT_TOKEN", "agent", "token", ""),
		MTLS: MTLSConfig{
			Enabled:    getValueBool("MTLS_ENABLED", "mtls", "enabled", false),
			ClientCert: getValue("CONTROL_CERT", "mtls", "client_cert", ""),
			ClientKey:  getValue("CONTROL_KEY", "mtls", "client_key", ""),
			CACert:     getValue("CONTROL_CA", "mtls", "ca_cert", ""),
		},
		RiskScanner: RiskScannerConfig{
			Enabled:               getValueBool("RISK_SCANNER_ENABLED", "risk_scanner", "enabled", true),
			IntervalSec:           getValueInt("RISK_SCANNER_INTERVAL_SEC", "risk_scanner", "interval_sec", 300),
			CertExpiringDays:      getValueInt("CERT_EXPIRING_DAYS", "risk_scanner", "cert_expiring_days", 15),
			CertExpiringThreshold: getValueInt("CERT_EXPIRING_WEBSITE_THRESHOLD", "risk_scanner", "cert_expiring_threshold", 2),
			ACMEMaxAttempts:       getValueInt("ACME_MAX_ATTEMPTS", "risk_scanner", "acme_max_attempts", 3),
		},
		ReleaseExecutor: ReleaseExecutorConfig{
			Enabled:     getValueBool("RELEASE_EXECUTOR_ENABLED", "release_executor", "enabled", true),
			IntervalSec: getValueInt("RELEASE_EXECUTOR_INTERVAL_SEC", "release_executor", "interval_sec", 5),
		},
		DNSWorker: DNSWorkerConfig{
			Enabled:     getValueBool("DNS_WORKER_ENABLED", "dns", "worker_enabled", true),
			IntervalSec: getValueInt("DNS_WORKER_INTERVAL_SEC", "dns", "interval_sec", 30),
			BatchSize:   getValueInt("DNS_WORKER_BATCH_SIZE", "dns", "batch_size", 10),
		},
ACMEWorker: ACMEWorkerConfig{
				Enabled:     getValueBool("ACME_WORKER_ENABLED", "acme", "worker_enabled", true),
				IntervalSec: getValueInt("ACME_WORKER_INTERVAL_SEC", "acme", "interval_sec", 40),
				BatchSize:   getValueInt("ACME_WORKER_BATCH_SIZE", "acme", "batch_size", 50),
			},
			CertCleaner: CertCleanerConfig{
				Enabled:        getValueBool("CERT_FAILED_CLEANER_ENABLED", "cert_cleaner", "enabled", true),
				IntervalSec:    getValueInt("CERT_FAILED_CLEANER_INTERVAL_SEC", "cert_cleaner", "interval_sec", 40),
				FailedKeepDays: getValueInt("CERT_FAILED_KEEP_DAYS", "cert_cleaner", "failed_keep_days", 3),
			},
			NodeHealthWorker: NodeHealthWorkerConfig{
				Enabled:              getValueBool("NODE_HEALTH_WORKER_ENABLED", "nodeHealthWorker", "enabled", true),
				IntervalSec:          getValueInt("NODE_HEALTH_WORKER_INTERVAL_SEC", "nodeHealthWorker", "intervalSec", 10),
				TimeoutSec:           getValueInt("NODE_HEALTH_WORKER_TIMEOUT_SEC", "nodeHealthWorker", "timeoutSec", 3),
				Concurrency:          getValueInt("NODE_HEALTH_WORKER_CONCURRENCY", "nodeHealthWorker", "concurrency", 10),
				OfflineFailThreshold: getValueInt("NODE_HEALTH_WORKER_OFFLINE_THRESHOLD", "nodeHealthWorker", "offlineFailThreshold", 2),
			},
		}

	// Validate required fields
	if cfg.MySQL.DSN == "" {
		return nil, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}
