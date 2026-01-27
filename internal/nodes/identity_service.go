package nodes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"go_cmdb/internal/model"
	"go_cmdb/internal/pki"
	"math/big"
	"net"
	"time"

	"gorm.io/gorm"
)

// IdentityService handles agent identity certificate generation and management
type IdentityService struct {
	db        *gorm.DB
	caManager *pki.CAManager
}

// NewIdentityService creates a new identity service
func NewIdentityService(db *gorm.DB, caManager *pki.CAManager) *IdentityService {
	return &IdentityService{
		db:        db,
		caManager: caManager,
	}
}

// GenerateCertificate generates a new mTLS client certificate for a node
// Returns cert PEM, key PEM, and fingerprint
func (s *IdentityService) GenerateCertificate(nodeID int, nodeName string, mainIP string) (certPEM, keyPEM, fingerprint string, err error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Prepare certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(3650 * 24 * time.Hour) // 10 years

	// Parse mainIP to add as SAN
	var ipAddresses []net.IP
	if mainIP != "" {
		if ip := net.ParseIP(mainIP); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("node-%d-%s", nodeID, nodeName),
			Organization: []string{"CDN Control Plane"},
		},
		IPAddresses:           ipAddresses,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the certificate with CA
	derBytes, err := s.caManager.SignCertificate(&template, &privateKey.PublicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Calculate fingerprint (SHA256 of DER bytes)
	hash := sha256.Sum256(derBytes)
	fingerprint = hex.EncodeToString(hash[:])

	// Encode certificate to PEM
	certPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	certPEM = string(certPEMBytes)

	// Encode private key to PEM
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	keyPEM = string(keyPEMBytes)

	return certPEM, keyPEM, fingerprint, nil
}

// CreateIdentity creates a new agent identity for a node
func (s *IdentityService) CreateIdentity(tx *gorm.DB, nodeID int, nodeName string, mainIP string) (*model.AgentIdentity, error) {
	// Check if identity already exists
	var existing model.AgentIdentity
	if err := tx.Where("node_id = ?", nodeID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("identity already exists for node %d", nodeID)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing identity: %w", err)
	}

	// Generate certificate
	certPEM, keyPEM, fingerprint, err := s.GenerateCertificate(nodeID, nodeName, mainIP)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	// Create identity record
	now := time.Now()
	identity := &model.AgentIdentity{
		NodeID:      nodeID,
		Fingerprint: fingerprint,
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		Status:      model.AgentIdentityStatusActive,
		IssuedAt:    &now,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	if err := tx.Create(identity).Error; err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	return identity, nil
}

// GetIdentityByNodeID retrieves the identity for a node
func (s *IdentityService) GetIdentityByNodeID(nodeID int) (*model.AgentIdentity, error) {
	var identity model.AgentIdentity
	if err := s.db.Where("node_id = ?", nodeID).First(&identity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("identity not found for node %d", nodeID)
		}
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}
	return &identity, nil
}

// RevokeIdentity revokes an agent identity
func (s *IdentityService) RevokeIdentity(nodeID int) error {
	now := time.Now()
	result := s.db.Model(&model.AgentIdentity{}).
		Where("node_id = ?", nodeID).
		Updates(map[string]interface{}{
			"status":     model.AgentIdentityStatusRevoked,
			"revoked_at": &now,
			"updated_at": &now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke identity: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("identity not found for node %d", nodeID)
	}

	return nil
}
