package acme

import "go_cmdb/internal/model"

// AcmeResult represents the result of a certificate request
type AcmeResult struct {
	CertPem  string   // Certificate PEM
	KeyPem   string   // Private key PEM
	ChainPem string   // Certificate chain PEM
	Issuer   string   // Certificate issuer
	Domains  []string // Domains covered by the certificate
}

// AcmeProvider defines the interface for ACME providers
type AcmeProvider interface {
	// EnsureAccount ensures an ACME account exists and is registered
	// If the account is not registered, it will register it with the ACME server
	EnsureAccount(account *model.AcmeAccount) error

	// RequestCertificate requests a certificate for the given domains
	// Returns the certificate result or an error
	RequestCertificate(domains []string) (*AcmeResult, error)
}
