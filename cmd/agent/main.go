package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	"go_cmdb/agent/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Get configuration
	httpAddr := getEnv("AGENT_HTTP_ADDR", ":9090")
	serverCert := getEnv("AGENT_CERT", "")
	serverKey := getEnv("AGENT_KEY", "")
	caCert := getEnv("AGENT_CA", "")

	log.Printf("Starting agent server with mTLS...")
	log.Printf("HTTPS address: %s", httpAddr)

	// Validate required mTLS configuration
	if serverCert == "" || serverKey == "" || caCert == "" {
		log.Fatal("AGENT_CERT, AGENT_KEY, and AGENT_CA are required for mTLS")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		log.Fatalf("Failed to load server certificate: %v", err)
	}

	// Calculate and print certificate fingerprint
	if len(cert.Certificate) > 0 {
		fingerprint := sha256.Sum256(cert.Certificate[0])
		fingerprintHex := hex.EncodeToString(fingerprint[:])
		log.Printf("Agent certificate fingerprint (SHA256): %s", fingerprintHex)
	}

	// Load CA certificate for client verification
	caCertBytes, err := os.ReadFile(caCert)
	if err != nil {
		log.Fatalf("Failed to load CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertBytes) {
		log.Fatal("Failed to append CA certificate")
	}

	// Create TLS config with client certificate verification
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // MUST verify client certificate
		MinVersion:   tls.VersionTLS12,
	}

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Setup routes (no token, mTLS only)
	v1.SetupRouter(r)

	// Create HTTPS server
	server := &http.Server{
		Addr:      httpAddr,
		Handler:   r,
		TLSConfig: tlsConfig,
	}

	log.Printf("Agent server running on %s (HTTPS with mTLS)", httpAddr)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Failed to start agent server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
