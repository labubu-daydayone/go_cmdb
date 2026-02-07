package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"go_cmdb/internal/dns"
	"go_cmdb/internal/domainutil"
	"go_cmdb/internal/model"

	"github.com/go-acme/lego/v4/certificate"
	legodns "github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"gorm.io/gorm"
)

// LegoClient implements AcmeProvider using go-acme/lego
type LegoClient struct {
	db               *gorm.DB
	dnsService       *dns.Service
	providerConfig   *model.AcmeProvider
	account          *model.AcmeAccount
	certificateRequestID int // For DNS challenge owner_id
}

// NewLegoClient creates a new lego client
func NewLegoClient(db *gorm.DB, dnsService *dns.Service, providerConfig *model.AcmeProvider, account *model.AcmeAccount, certRequestID int) *LegoClient {
	return &LegoClient{
		db:               db,
		dnsService:       dnsService,
		providerConfig:   providerConfig,
		account:          account,
		certificateRequestID: certRequestID,
	}
}

// User implements registration.User interface for lego
type User struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// EnsureAccount ensures an ACME account exists and is registered
func (c *LegoClient) EnsureAccount(account *model.AcmeAccount) error {
	// If account is already active, skip registration
	if account.Status == model.AcmeAccountStatusActive && account.RegistrationURI != "" {
		return nil
	}

	// Parse or generate account private key
	var privateKey crypto.PrivateKey
	var err error

	if account.AccountKeyPem != "" {
		// Parse existing key
		privateKey, err = parsePrivateKey(account.AccountKeyPem)
		if err != nil {
			return fmt.Errorf("failed to parse account key: %w", err)
		}
	} else {
		// Generate new key
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return fmt.Errorf("failed to generate account key: %w", err)
		}

		// Save key to account
		keyPem, err := encodePrivateKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to encode account key: %w", err)
		}
		account.AccountKeyPem = keyPem
	}

	// Create lego config
	config := lego.NewConfig(&User{
		Email: account.Email,
		key:   privateKey,
	})
	config.CADirURL = c.providerConfig.DirectoryURL

	// Create lego client
	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create lego client: %w", err)
	}

	// Register account
	var reg *registration.Resource
	if c.providerConfig.RequiresEAB {
		// External Account Binding (Google Public CA)
		if account.EabKid == "" || account.EabHmacKey == "" {
			return errors.New("EAB credentials required but not provided")
		}
		reg, err = client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
			TermsOfServiceAgreed: true,
			Kid:                  account.EabKid,
			HmacEncoded:          account.EabHmacKey,
		})
	} else {
		// Standard registration (Let's Encrypt)
		reg, err = client.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to register ACME account: %w", err)
	}

	// Save registration URI
	account.RegistrationURI = reg.URI
	account.Status = model.AcmeAccountStatusActive

	// Update account in database
	if err := c.db.Save(account).Error; err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	return nil
}

// RequestCertificate requests a certificate for the given domains
func (c *LegoClient) RequestCertificate(domains []string) (*AcmeResult, error) {
	// Parse account private key
	privateKey, err := parsePrivateKey(c.account.AccountKeyPem)
	if err != nil {
		return nil, fmt.Errorf("failed to parse account key: %w", err)
	}

	// Create lego user
	user := &User{
		Email: c.account.Email,
		Registration: &registration.Resource{
			URI: c.account.RegistrationURI,
		},
		key: privateKey,
	}

	// Create lego config
	config := lego.NewConfig(user)
	config.CADirURL = c.providerConfig.DirectoryURL

	// Create lego client
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	// Set up DNS-01 challenge provider
	dnsProvider := &CustomDNSProvider{
		dnsService: c.dnsService,
		certRequestID: c.certificateRequestID,
	}

	err = client.Challenge.SetDNS01Provider(dnsProvider,
		legodns.AddRecursiveNameservers([]string{"8.8.8.8:53", "1.1.1.1:53"}),
		legodns.WrapPreCheck(func(domain, fqdn, value string, check legodns.PreCheckFunc) (bool, error) {
			// Wait for DNS propagation (DNS Worker should have synced the TXT record)
			time.Sleep(30 * time.Second)
			return check(fqdn, value)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set DNS provider: %w", err)
	}

	// Request certificate
	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Parse certificate to get issuer
	certBlock, _ := pem.Decode(certificates.Certificate)
	if certBlock == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &AcmeResult{
		CertPem:  string(certificates.Certificate),
		KeyPem:   string(certificates.PrivateKey),
		ChainPem: string(certificates.IssuerCertificate),
		Issuer:   cert.Issuer.CommonName,
		Domains:  domains,
	}, nil
}

// CustomDNSProvider implements challenge.Provider for DNS-01 challenge
type CustomDNSProvider struct {
	dnsService    *dns.Service
	certRequestID int
}

// Present creates a TXT record for DNS-01 challenge
func (p *CustomDNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := legodns.GetRecord(domain, keyAuth)

	// Extract domain name from FQDN
	// fqdn is like "_acme-challenge.example.com."
	// We need to extract apex (e.g. "example.com") and subdomain (e.g. "_acme-challenge")
	cleanFQDN := strings.TrimSuffix(fqdn, ".")

	// Use PSL to calculate apex (eTLD+1)
	baseDomain, err := domainutil.EffectiveApex(cleanFQDN)
	if err != nil {
		return fmt.Errorf("failed to calculate apex for %s: %w", cleanFQDN, err)
	}

	// Get subdomain by removing the base domain suffix
	var subdomain string
	if len(cleanFQDN) > len(baseDomain)+1 {
		subdomain = cleanFQDN[:len(cleanFQDN)-len(baseDomain)-1]
	} else {
		subdomain = ""
	}

	// Find domain in database
	var domainRecord model.Domain
		if err := p.dnsService.GetDB().Where("domain = ?", baseDomain).First(&domainRecord).Error; err != nil {
		return fmt.Errorf("domain not found: %s", baseDomain)
	}

	// Create DNS TXT record via DNS service
	dnsRecord := &model.DomainDNSRecord{
		DomainID:     domainRecord.ID,
		Type:         model.DNSRecordTypeTXT,
		Name:         subdomain,
		Value:        value,
		TTL:          60,
		Status:       model.DNSRecordStatusPending,
		DesiredState: model.DNSRecordDesiredStatePresent,
		OwnerType:    "acme_challenge",
		OwnerID:      p.certRequestID,
	}

	if err := p.dnsService.GetDB().Create(dnsRecord).Error; err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	// Wait for DNS Worker to sync (40 seconds + buffer)
	time.Sleep(50 * time.Second)

	return nil
}

// CleanUp removes the TXT record after challenge completion
func (p *CustomDNSProvider) CleanUp(domain, token, keyAuth string) error {
	// Mark DNS records for deletion (desired_state=absent)
	// DNS Worker will handle the actual deletion
	return p.dnsService.GetDB().
		Model(&model.DomainDNSRecord{}).
		Where("owner_type = ? AND owner_id = ?", "acme_challenge", p.certRequestID).
		Update("desired_state", model.DNSRecordDesiredStateAbsent).
		Error
}

// parsePrivateKey parses a PEM-encoded private key
func parsePrivateKey(keyPem string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(keyPem))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Try EC private key
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// Try PKCS8 private key
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.New("unsupported private key type")
}

// encodePrivateKey encodes a private key to PEM format
func encodePrivateKey(key crypto.PrivateKey) (string, error) {
	var keyBytes []byte
	var err error

	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		keyBytes, err = x509.MarshalECPrivateKey(k)
		if err != nil {
			return "", err
		}
	default:
		return "", errors.New("unsupported private key type")
	}

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	return string(pem.EncodeToMemory(block)), nil
}
