package main

import (
	"log"
	"os"

	"go_cmdb/agent/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Get configuration
	agentToken := getEnv("AGENT_TOKEN", "default-agent-token")
	httpAddr := getEnv("AGENT_HTTP_ADDR", ":9090")

	log.Printf("Starting agent server...")
	log.Printf("Agent token: %s", agentToken)
	log.Printf("HTTP address: %s", httpAddr)

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Setup routes
	v1.SetupRouter(r, agentToken)

	// Start server
	log.Printf("Agent server running on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatalf("Failed to start agent server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
