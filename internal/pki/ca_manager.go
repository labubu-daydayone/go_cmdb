package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CAManager manages the root CA certificate and key
type CAManager struct {
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	caCertPEM string
	caKeyPEM  string
	mu        sync.RWMutex
	certPath  string
	keyPath   string
}

// NewCAManager creates a new CA manager
// It will load existing CA from disk or generate a new one
func NewCAManager(dataDir string) (*CAManager, error) {
	certPath := filepath.Join(dataDir, "ca.crt")
	keyPath := filepath.Join(dataDir, "ca.key")

	manager := &CAManager{
		certPath: certPath,
		keyPath:  keyPath,
	}

	// Try to load existing CA
	if err := manager.loadCA(); err == nil {
		return manager, nil
	}

	// Generate new CA if not exists
	if err := manager.generateCA(); err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	// Save CA to disk
	if err := manager.saveCA(); err != nil {
		return nil, fmt.Errorf("failed to save CA: %w", err)
	}

	return manager, nil
}

// loadCA loads CA from disk
func (m *CAManager) loadCA() error {
	// Read cert file
	certPEM, err := os.ReadFile(m.certPath)
	if err != nil {
		return fmt.Errorf("failed to read CA cert: %w", err)
	}

	// Read key file
	keyPEM, err := os.ReadFile(m.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	// Parse cert
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA cert PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA cert: %w", err)
	}

	// Parse key
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA key: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.caCert = caCert
	m.caKey = caKey
	m.caCertPEM = string(certPEM)
	m.caKeyPEM = string(keyPEM)

	return nil
}

// generateCA generates a new CA certificate and key
func (m *CAManager) generateCA() error {
	// Generate RSA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %w", err)
	}

	// Prepare CA certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(3650 * 24 * time.Hour) // 10 years

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "CDN Control Plane CA",
			Organization: []string{"CDN Control Plane"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign the CA certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse the generated certificate
	caCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// Encode private key to PEM
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caKey),
	})

	m.mu.Lock()
	defer m.mu.Unlock()

	m.caCert = caCert
	m.caKey = caKey
	m.caCertPEM = string(certPEMBytes)
	m.caKeyPEM = string(keyPEMBytes)

	return nil
}

// saveCA saves CA to disk
func (m *CAManager) saveCA() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(m.certPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Write cert file
	if err := os.WriteFile(m.certPath, []byte(m.caCertPEM), 0644); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}

	// Write key file (with restricted permissions)
	if err := os.WriteFile(m.keyPath, []byte(m.caKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	return nil
}

// GetCACertPEM returns the CA certificate in PEM format
func (m *CAManager) GetCACertPEM() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.caCertPEM
}

// SignCertificate signs a client certificate using the CA
func (m *CAManager) SignCertificate(template *x509.Certificate, publicKey *rsa.PublicKey) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.caCert == nil || m.caKey == nil {
		return nil, fmt.Errorf("CA not initialized")
	}

	// Sign the certificate with CA
	derBytes, err := x509.CreateCertificate(rand.Reader, template, m.caCert, publicKey, m.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return derBytes, nil
}
