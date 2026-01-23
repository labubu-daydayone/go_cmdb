package render

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"go_cmdb/agent/config"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Renderer handles template rendering
type Renderer struct {
	dirConfig *config.DirConfig
	templates *template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer(dirConfig *config.DirConfig) (*Renderer, error) {
	// Parse templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Renderer{
		dirConfig: dirConfig,
		templates: tmpl,
	}, nil
}

// UpstreamData holds data for upstream template
type UpstreamData struct {
	WebsiteID    int
	UpstreamName string
	Addresses    []AddressData
	GeneratedAt  string
}

// AddressData holds data for an upstream address
type AddressData struct {
	Role     string
	Protocol string
	Address  string
	Weight   int
	Enabled  bool
}

// ServerData holds data for server template
type ServerData struct {
	WebsiteID   int
	Domains     []DomainData
	Origin      OriginData
	HTTPS       HTTPSData
	CertPath    string
	KeyPath     string
	GeneratedAt string
}

// DomainData holds data for a domain
type DomainData struct {
	Domain    string
	IsPrimary bool
}

// OriginData holds data for origin configuration
type OriginData struct {
	Mode               string
	RedirectURL        string
	RedirectStatusCode int
	UpstreamName       string
}

// HTTPSData holds data for HTTPS configuration
type HTTPSData struct {
	Enabled       bool
	ForceRedirect bool
	HSTS          bool
}

// RenderUpstream renders upstream configuration to a file
func (r *Renderer) RenderUpstream(stagingDir string, websiteID int, data *UpstreamData) error {
	// Set generated time
	data.GeneratedAt = time.Now().Format(time.RFC3339)

	// Render template
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, "upstream.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute upstream template: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("upstream_site_%d.conf", websiteID)
	filepath := filepath.Join(stagingDir, "upstreams", filename)

	if err := os.WriteFile(filepath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write upstream file: %w", err)
	}

	return nil
}

// RenderServer renders server configuration to a file
func (r *Renderer) RenderServer(stagingDir string, websiteID int, data *ServerData) error {
	// Set generated time
	data.GeneratedAt = time.Now().Format(time.RFC3339)

	// Render template
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, "server.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute server template: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("server_site_%d.conf", websiteID)
	filepath := filepath.Join(stagingDir, "servers", filename)

	if err := os.WriteFile(filepath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write server file: %w", err)
	}

	return nil
}

// WriteCertificate writes certificate and key files
func (r *Renderer) WriteCertificate(stagingDir string, certificateID int, certPem, keyPem string) error {
	certsDir := filepath.Join(stagingDir, "certs")

	// Write certificate
	certPath := filepath.Join(certsDir, fmt.Sprintf("cert_%d.pem", certificateID))
	if err := os.WriteFile(certPath, []byte(certPem), 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write key
	keyPath := filepath.Join(certsDir, fmt.Sprintf("key_%d.pem", certificateID))
	if err := os.WriteFile(keyPath, []byte(keyPem), 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}
