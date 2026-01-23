package main

import (
	"log"
	"os"

	"go_cmdb/api/v1"
	"go_cmdb/internal/auth"
	"go_cmdb/internal/cache"
	"go_cmdb/internal/config"
	"go_cmdb/internal/db"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		os.Exit(1)
	}
	log.Println("✓ Configuration loaded")

	// 2. Initialize MySQL
	if err := db.InitMySQL(cfg.MySQL.DSN); err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Println("✓ MySQL connected successfully")

	// Auto migrate user table
	if err := db.GetDB().AutoMigrate(&model.User{}); err != nil {
		log.Printf("Warning: Failed to auto migrate: %v", err)
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

	// 5. Initialize Gin router
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
