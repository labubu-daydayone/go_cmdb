package cert

import (
	"fmt"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service handles certificate-related business logic
type Service struct {
	db *gorm.DB
}

// NewService creates a new certificate service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// WebsiteInfo represents website information for certificate binding
type WebsiteInfo struct {
	WebsiteID      int      `json:"websiteId"`
	PrimaryDomain  string   `json:"primaryDomain"`
	Domains        []string `json:"domains"`
	HTTPSEnabled   bool     `json:"httpsEnabled"`
	BindStatus     string   `json:"bindStatus"` // active / inactive
}

// GetCertificateWebsites returns all websites using a certificate
func (s *Service) GetCertificateWebsites(certificateID int) ([]WebsiteInfo, error) {
	// Get all bindings for this certificate
	var bindings []model.CertificateBinding
	if err := s.db.Where("certificate_id = ?", certificateID).Find(&bindings).Error; err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return []WebsiteInfo{}, nil
	}

	// Extract website IDs
	websiteIDs := make([]int, len(bindings))
	bindingMap := make(map[int]string) // website_id -> status
	for i, b := range bindings {
		websiteIDs[i] = b.WebsiteID
		bindingMap[b.WebsiteID] = b.Status
	}

	// Get websites
	var websites []model.Website
	if err := s.db.Where("id IN ?", websiteIDs).Find(&websites).Error; err != nil {
		return nil, err
	}

	// Build result
	result := make([]WebsiteInfo, len(websites))
	for i, w := range websites {
		// Get website domains
		domains, err := s.GetWebsiteDomains(w.ID)
		if err != nil {
			domains = []string{}
		}

		// Get primary domain (first domain)
		primaryDomain := ""
		if len(domains) > 0 {
			primaryDomain = domains[0]
		}

		// Get HTTPS status
		var httpsRecord model.WebsiteHTTPS
		httpsEnabled := false
		if err := s.db.Where("website_id = ?", w.ID).First(&httpsRecord).Error; err == nil {
			httpsEnabled = httpsRecord.Enabled
		}

		result[i] = WebsiteInfo{
			WebsiteID:     w.ID,
			PrimaryDomain: primaryDomain,
			Domains:       domains,
			HTTPSEnabled:  httpsEnabled,
			BindStatus:    bindingMap[w.ID],
		}
	}

	return result, nil
}

// GetCertificateDomains returns all domains for a certificate
func (s *Service) GetCertificateDomains(certificateID int) ([]string, error) {
	var certDomains []model.CertificateDomain
	if err := s.db.Where("certificate_id = ?", certificateID).Find(&certDomains).Error; err != nil {
		return nil, err
	}

	domains := make([]string, len(certDomains))
	for i, cd := range certDomains {
		domains[i] = cd.Domain
	}

	return domains, nil
}

// GetWebsiteDomains returns all domains for a website
func (s *Service) GetWebsiteDomains(websiteID int) ([]string, error) {
	var websiteDomains []model.WebsiteDomain
	if err := s.db.Where("website_id = ?", websiteID).Find(&websiteDomains).Error; err != nil {
		return nil, err
	}

	domains := make([]string, len(websiteDomains))
	for i, wd := range websiteDomains {
		domains[i] = wd.Domain
	}

	return domains, nil
}

// CertificateCandidate represents a certificate candidate for a website
type CertificateCandidate struct {
	CertificateID      int            `json:"certificateId"`
	CertificateName    string         `json:"certificateName"`
	CertificateDomains []string       `json:"certificateDomains"`
	CoverageStatus     CoverageStatus `json:"coverageStatus"`
	MissingDomains     []string       `json:"missingDomains,omitempty"`
	ExpireAt           string         `json:"expireAt"`
	Provider           string         `json:"provider"`
}

// GetWebsiteCertificateCandidates returns all certificate candidates for a website
func (s *Service) GetWebsiteCertificateCandidates(websiteID int) ([]CertificateCandidate, error) {
	// Get website domains
	websiteDomains, err := s.GetWebsiteDomains(websiteID)
	if err != nil {
		return nil, err
	}

	if len(websiteDomains) == 0 {
		return []CertificateCandidate{}, nil
	}

	// Get all certificates
	var certificates []model.Certificate
	if err := s.db.Find(&certificates).Error; err != nil {
		return nil, err
	}

	// Calculate coverage for each certificate
	candidates := make([]CertificateCandidate, 0, len(certificates))
	for _, cert := range certificates {
		// Get certificate domains
		certDomains, err := s.GetCertificateDomains(cert.ID)
		if err != nil {
			continue
		}

		// Calculate coverage
		coverage := CalculateCoverage(certDomains, websiteDomains)

		// Determine provider
		provider := "Manual"
		if cert.Source == model.CertificateSourceAcme {
			provider = "ACME"
		}

			candidates = append(candidates, CertificateCandidate{
				CertificateID:      cert.ID,
				CertificateName:    fmt.Sprintf("Certificate #%d", cert.ID),
				CertificateDomains: certDomains,
				CoverageStatus:     coverage.Status,
				MissingDomains:     coverage.MissingDomains,
				ExpireAt:           cert.ExpireAt.Format("2006-01-02 15:04:05"),
				Provider:           provider,
			})
	}

	return candidates, nil
}

// ValidateCertificateCoverage validates if a certificate covers all website domains
// Returns error if coverage is not complete
func (s *Service) ValidateCertificateCoverage(certificateID int, websiteID int) (*CoverageResult, error) {
	// Get certificate domains
	certDomains, err := s.GetCertificateDomains(certificateID)
	if err != nil {
		return nil, err
	}

	// Get website domains
	websiteDomains, err := s.GetWebsiteDomains(websiteID)
	if err != nil {
		return nil, err
	}

	// Calculate coverage
	coverage := CalculateCoverage(certDomains, websiteDomains)

	return &coverage, nil
}

// CertificateInfo represents parsed certificate information
type CertificateInfo struct {
	CommonName  string
	Fingerprint string
	Issuer      string
	IssueAt     string
	ExpireAt    string
	Status      string
}

// ParseCertificate parses a PEM-encoded certificate and extracts metadata
func (s *Service) ParseCertificate(certPem string) (*CertificateInfo, error) {
	// TODO: Implement actual certificate parsing using crypto/x509
	// For now, return a placeholder implementation
	// This should:
	// 1. Decode PEM block
	// 2. Parse X.509 certificate
	// 3. Extract CommonName, Issuer, NotBefore, NotAfter
	// 4. Calculate SHA256 fingerprint
	// 5. Determine status based on expiration date
	
	// Placeholder implementation
	return &CertificateInfo{
		CommonName:  "placeholder.com",
		Fingerprint: "placeholder_fingerprint",
		Issuer:      "Placeholder CA",
		IssueAt:     "2026-01-01 00:00:00",
		ExpireAt:    "2027-01-01 00:00:00",
		Status:      "valid",
	}, nil
}

// ValidatePrivateKey validates a PEM-encoded private key
func (s *Service) ValidatePrivateKey(keyPem string) error {
	// TODO: Implement actual private key validation using crypto/x509
	// This should:
	// 1. Decode PEM block
	// 2. Parse private key (RSA/ECDSA)
	// 3. Validate key format and structure
	
	// Placeholder implementation
	if keyPem == "" {
		return gorm.ErrInvalidData
	}
	return nil
}
