package main

import (
	"flag"
	"log"
	"os"

	"context"
	"time"

	"go_cmdb/api/v1"
	"go_cmdb/internal/agentclient"
	"go_cmdb/internal/auth"
	"go_cmdb/internal/cache"
	"go_cmdb/internal/config"
	"go_cmdb/internal/db"
	"go_cmdb/internal/dns"
	"go_cmdb/internal/release"
	"go_cmdb/internal/risk"
	"go_cmdb/internal/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config.ini file")
	flag.Parse()

	// 1. Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		log.Printf("Loading configuration from INI file: %s", *configPath)
		cfg, err = config.LoadFromINI(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config from INI: %v", err)
			os.Exit(1)
		}
		log.Println("✓ Configuration loaded from INI file")
	} else {
		log.Println("Loading configuration from environment variables")
		cfg, err = config.Load()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
			os.Exit(1)
		}
		log.Println("✓ Configuration loaded from environment")
	}

	// 2. Initialize MySQL
	if err := db.InitMySQL(cfg.MySQL.DSN); err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Println("✓ MySQL connected successfully")

	// Run database migration if MIGRATE=1
	if cfg.Migrate {
		log.Println("MIGRATE=1 detected, running database migration...")
		if err := db.Migrate(db.GetDB()); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
			os.Exit(1)
		}
	} else {
		log.Println("MIGRATE=0 or not set, migration disabled")
	}

	// 3. Initialize JWT
	auth.InitJWT(cfg.JWT.Secret)
	log.Println("✓ JWT initialized")

	// 4. Initialize Redis
	if err := cache.InitRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
		os.Exit(1)
	}
	defer cache.Close()
	log.Println("✓ Redis connected successfully")

	// 5. Start Risk Scanner
	scannerConfig := risk.ScannerConfig{
		Enabled:              cfg.RiskScanner.Enabled,
		IntervalSec:          cfg.RiskScanner.IntervalSec,
		CertExpiringDays:     cfg.RiskScanner.CertExpiringDays,
		CertExpiringThreshold: cfg.RiskScanner.CertExpiringThreshold,
		ACMEMaxAttempts:      cfg.RiskScanner.ACMEMaxAttempts,
	}
	scanner := risk.NewScanner(db.GetDB(), scannerConfig)
	scanner.Start()
	defer scanner.Stop()
	log.Println("✓ Risk Scanner initialized")

	// 6. Start Release Executor
	if cfg.ReleaseExecutor.Enabled {
		if !cfg.MTLS.Enabled {
			log.Println("⚠ Release Executor requires mTLS but mTLS is not enabled, skipping executor")
		} else {
			agentClient, err := agentclient.NewClient(cfg)
			if err != nil {
				log.Fatalf("Failed to create agent client: %v", err)
				os.Exit(1)
			}

			executor := release.NewExecutor(
				db.GetDB(),
				agentClient,
				time.Duration(cfg.ReleaseExecutor.IntervalSec)*time.Second,
			)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go executor.RunLoop(ctx)
			log.Println("✓ Release Executor initialized")
		}
	} else {
		log.Println("✓ Release Executor disabled (RELEASE_EXECUTOR_ENABLED=0)")
	}

	// 7. Start DNS Worker
	if cfg.DNSWorker.Enabled {
		workerConfig := dns.WorkerConfig{
			Enabled:     cfg.DNSWorker.Enabled,
			IntervalSec: cfg.DNSWorker.IntervalSec,
			BatchSize:   cfg.DNSWorker.BatchSize,
		}
		worker := dns.NewWorker(db.GetDB(), workerConfig)
		worker.Start()
		defer worker.Stop()
		log.Println("✓ DNS Worker initialized")
	} else {
		log.Println("✓ DNS Worker disabled (DNS_WORKER_ENABLED=0)")
	}

	// 8. Initialize Socket.IO server
	if err := ws.InitServer(); err != nil {
		log.Fatalf("Failed to initialize Socket.IO server: %v", err)
		os.Exit(1)
	}
	log.Println("✓ Socket.IO server initialized")

	// 9. Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Setup API v1 routes
	v1.SetupRouter(r, db.GetDB(), cfg)

	log.Printf("✓ Server starting on %s", cfg.HTTPAddr)

	// Start server
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
