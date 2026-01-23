package configgen

// ApplyConfigPayload represents the complete payload for apply_config task
type ApplyConfigPayload struct {
	Version  int64             `json:"version"`
	Websites []WebsiteConfig `json:"websites"`
}

// WebsiteConfig represents a single website configuration
type WebsiteConfig struct {
	WebsiteID int              `json:"websiteId"`
	Status    string           `json:"status"`
	Domains   []DomainConfig   `json:"domains"`
	Origin    OriginConfig     `json:"origin"`
	HTTPS     HTTPSConfig      `json:"https"`
}

// DomainConfig represents a domain configuration
type DomainConfig struct {
	Domain    string `json:"domain"`
	IsPrimary bool   `json:"isPrimary"`
	CNAME     string `json:"cname"`
}

// OriginConfig represents origin configuration
type OriginConfig struct {
	Mode               string            `json:"mode"` // group|manual|redirect
	RedirectURL        string            `json:"redirectUrl,omitempty"`
	RedirectStatusCode int               `json:"redirectStatusCode,omitempty"`
	UpstreamName       string            `json:"upstreamName,omitempty"`
	Addresses          []AddressConfig   `json:"addresses,omitempty"`
}

// AddressConfig represents an origin address
type AddressConfig struct {
	Role     string `json:"role"`     // primary|backup
	Protocol string `json:"protocol"` // http|https
	Address  string `json:"address"`  // ip:port or domain:port
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
}

// HTTPSConfig represents HTTPS configuration
type HTTPSConfig struct {
	Enabled       bool                `json:"enabled"`
	ForceRedirect bool                `json:"forceRedirect"`
	HSTS          bool                `json:"hsts"`
	Certificate   *CertificateConfig  `json:"certificate,omitempty"`
}

// CertificateConfig represents certificate configuration
type CertificateConfig struct {
	CertificateID int    `json:"certificateId"`
	CertPem       string `json:"certPem"`
	KeyPem        string `json:"keyPem"`
}
