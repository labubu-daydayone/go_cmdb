package configgen

import (
	"encoding/json"
	"fmt"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Aggregator aggregates configuration data for a node
type Aggregator struct {
	db *gorm.DB
}

// NewAggregator creates a new configuration aggregator
func NewAggregator(db *gorm.DB) *Aggregator {
	return &Aggregator{db: db}
}

// GeneratePayload generates the complete apply_config payload for a node
func (a *Aggregator) GeneratePayload(nodeID int, version int64) (*ApplyConfigPayload, error) {
	// Get all active websites
	var websites []model.Website
	if err := a.db.Where("status = ?", model.WebsiteStatusActive).Find(&websites).Error; err != nil {
		return nil, fmt.Errorf("failed to query websites: %w", err)
	}

	payload := &ApplyConfigPayload{
		Version:  version,
		Websites: make([]WebsiteConfig, 0, len(websites)),
	}

	for _, website := range websites {
		websiteConfig, err := a.buildWebsiteConfig(&website)
		if err != nil {
			return nil, fmt.Errorf("failed to build config for website %d: %w", website.ID, err)
		}

		payload.Websites = append(payload.Websites, *websiteConfig)
	}

	return payload, nil
}

// buildWebsiteConfig builds configuration for a single website
func (a *Aggregator) buildWebsiteConfig(website *model.Website) (*WebsiteConfig, error) {
	config := &WebsiteConfig{
		WebsiteID: website.ID,
		Status:    website.Status,
	}

	// Build domains
	domains, err := a.buildDomains(website.ID)
	if err != nil {
		return nil, err
	}
	config.Domains = domains

	// Build origin
	origin, err := a.buildOrigin(website)
	if err != nil {
		return nil, err
	}
	config.Origin = *origin

	// Build HTTPS
	https, err := a.buildHTTPS(website.ID)
	if err != nil {
		return nil, err
	}
	config.HTTPS = *https

	return config, nil
}

// buildDomains builds domain configuration
func (a *Aggregator) buildDomains(websiteID int) ([]DomainConfig, error) {
	var domains []model.WebsiteDomain
	if err := a.db.Where("website_id = ?", websiteID).Find(&domains).Error; err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}

	configs := make([]DomainConfig, 0, len(domains))
	for _, domain := range domains {
		configs = append(configs, DomainConfig{
			Domain:    domain.Domain,
			IsPrimary: domain.IsPrimary,
		})
	}

	return configs, nil
}

// buildOrigin builds origin configuration
func (a *Aggregator) buildOrigin(website *model.Website) (*OriginConfig, error) {
	config := &OriginConfig{
		Mode: website.OriginMode,
	}

	switch website.OriginMode {
	case model.OriginModeRedirect:
		config.RedirectURL = website.RedirectURL
		config.RedirectStatusCode = website.RedirectStatusCode
		return config, nil

	case model.OriginModeGroup, model.OriginModeManual:
		// Get origin set (snapshot)
		var originSet model.OriginSet
		if err := a.db.Where("website_id = ?", website.ID).
			Order("snapshot_at DESC").
			First(&originSet).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// No origin set yet, return empty addresses
				// 如果没有 origin_set，不生成 upstream
				config.Addresses = []AddressConfig{}
				return config, nil
			}
			return nil, fmt.Errorf("failed to query origin set: %w", err)
		}

		// Get origin addresses
		var addresses []model.OriginAddress
		if err := a.db.Where("origin_set_id = ?", originSet.ID).
			Order("role, id").
			Find(&addresses).Error; err != nil {
			return nil, fmt.Errorf("failed to query origin addresses: %w", err)
		}

			// upstream 命名改为以 origin_set_id 为键
			config.UpstreamName = fmt.Sprintf("upstream_originset_%d", originSet.ID)
			config.Addresses = make([]AddressConfig, 0, len(addresses))

		for _, addr := range addresses {
			config.Addresses = append(config.Addresses, AddressConfig{
				Role:     addr.Role,
				Protocol: addr.Protocol,
				Address:  addr.Address,
				Weight:   addr.Weight,
				Enabled:  addr.Enabled,
			})
		}

		return config, nil

	default:
		return nil, fmt.Errorf("unknown origin mode: %s", website.OriginMode)
	}
}

// buildHTTPS builds HTTPS configuration
func (a *Aggregator) buildHTTPS(websiteID int) (*HTTPSConfig, error) {
	// Get website HTTPS config
	var websiteHTTPS model.WebsiteHTTPS
	if err := a.db.Where("website_id = ?", websiteID).First(&websiteHTTPS).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No HTTPS config, return disabled
			return &HTTPSConfig{
				Enabled: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to query website HTTPS: %w", err)
	}

	config := &HTTPSConfig{
		Enabled:       websiteHTTPS.Enabled,
		ForceRedirect: websiteHTTPS.ForceRedirect,
		HSTS:          websiteHTTPS.HSTS,
	}

	// If HTTPS is disabled, no need to load certificate
	if !websiteHTTPS.Enabled {
		return config, nil
	}

	// Load certificate if cert_mode is select
	if websiteHTTPS.CertMode == model.CertModeSelect && websiteHTTPS.CertificateID != nil && *websiteHTTPS.CertificateID > 0 {
		var certificate model.Certificate
		if err := a.db.First(&certificate, websiteHTTPS.CertificateID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Certificate not found, disable HTTPS
				// (Alternative: return error, but we choose to gracefully degrade)
				config.Enabled = false
				return config, nil
			}
			return nil, fmt.Errorf("failed to query certificate: %w", err)
		}

		config.Certificate = &CertificateConfig{
			CertificateID: certificate.ID,
			CertPem:       certificate.CertificatePem,
			KeyPem:        certificate.PrivateKeyPem,
		}
	} else if websiteHTTPS.CertMode == model.CertModeACME {
		// ACME mode: check if certificate is issued
		// For now, we disable HTTPS if certificate is not ready
		// (Future: implement ACME worker to auto-issue certificates)
		if websiteHTTPS.CertificateID == nil || *websiteHTTPS.CertificateID == 0 {
			config.Enabled = false
			return config, nil
		}

		// Load ACME certificate
		var certificate model.Certificate
		if err := a.db.First(&certificate, websiteHTTPS.CertificateID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				config.Enabled = false
				return config, nil
			}
			return nil, fmt.Errorf("failed to query ACME certificate: %w", err)
		}

		// Check if certificate is issued
		if certificate.Status != model.CertificateStatusIssued {
			config.Enabled = false
			return config, nil
		}

		config.Certificate = &CertificateConfig{
			CertificateID: certificate.ID,
			CertPem:       certificate.CertificatePem,
			KeyPem:        certificate.PrivateKeyPem,
		}
	}

	return config, nil
}

// SerializePayload serializes the payload to JSON string
func (a *Aggregator) SerializePayload(payload *ApplyConfigPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to serialize payload: %w", err)
	}

	return string(data), nil
}
