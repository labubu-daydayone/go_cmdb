package nodehealth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"
)

// NewMTLSClient creates a new HTTP client configured for mTLS.
func NewMTLSClient(caCertPath, clientCertPath, clientKeyPath string, timeout time.Duration) (*http.Client, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	block, _ := pem.Decode(caCert)
	if block == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM block from %s", caCertPath)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Load client certificate
	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client key pair: %w", err)
	}

	// Validate client certificate and key before use
	if _, err := x509.ParseCertificate(clientCert.Certificate[0]); err != nil {
		return nil, fmt.Errorf("failed to parse client certificate from %s: %w", clientCertPath, err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
	}

	// Create HTTP transport
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Create HTTP client
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return client, nil
}
