package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go_cmdb/agent/config"
	"go_cmdb/agent/render"
)

// ApplyConfigPayload represents the payload for apply_config task
type ApplyConfigPayload struct {
	Version  int64           `json:"version"`
	Websites []WebsiteConfig `json:"websites"`
}

// WebsiteConfig represents a website configuration
type WebsiteConfig struct {
	WebsiteID int            `json:"websiteId"`
	Status    string         `json:"status"`
	Domains   []DomainConfig `json:"domains"`
	Origin    OriginConfig   `json:"origin"`
	HTTPS     HTTPSConfig    `json:"https"`
}

// DomainConfig represents a domain configuration
type DomainConfig struct {
	Domain    string `json:"domain"`
	IsPrimary bool   `json:"isPrimary"`
	CNAME     string `json:"cname"`
}

// OriginConfig represents origin configuration
type OriginConfig struct {
	Mode               string          `json:"mode"`
	RedirectURL        string          `json:"redirectUrl,omitempty"`
	RedirectStatusCode int             `json:"redirectStatusCode,omitempty"`
	UpstreamName       string          `json:"upstreamName,omitempty"`
	Addresses          []AddressConfig `json:"addresses,omitempty"`
}

// AddressConfig represents an origin address
type AddressConfig struct {
	Role     string `json:"role"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
}

// HTTPSConfig represents HTTPS configuration
type HTTPSConfig struct {
	Enabled       bool               `json:"enabled"`
	ForceRedirect bool               `json:"forceRedirect"`
	HSTS          bool               `json:"hsts"`
	Certificate   *CertificateConfig `json:"certificate,omitempty"`
}

// CertificateConfig represents certificate configuration
type CertificateConfig struct {
	CertificateID int    `json:"certificateId"`
	CertPem       string `json:"certPem"`
	KeyPem        string `json:"keyPem"`
}

// VersionMeta represents version metadata
type VersionMeta struct {
	Version   int64  `json:"version"`
	AppliedAt string `json:"appliedAt"`
}

// ErrorMeta represents error metadata
type ErrorMeta struct {
	Version int64  `json:"version"`
	Error   string `json:"error"`
	Time    string `json:"time"`
}

// ApplyConfigExecutor handles apply_config task execution
type ApplyConfigExecutor struct {
	dirConfig *config.DirConfig
	renderer  *render.Renderer
}

// NewApplyConfigExecutor creates a new apply_config executor
func NewApplyConfigExecutor(dirConfig *config.DirConfig) (*ApplyConfigExecutor, error) {
	renderer, err := render.NewRenderer(dirConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	return &ApplyConfigExecutor{
		dirConfig: dirConfig,
		renderer:  renderer,
	}, nil
}

// Execute executes the apply_config task
func (e *ApplyConfigExecutor) Execute(payloadJSON string) (string, error) {
	// Parse payload
	var payload ApplyConfigPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return "", fmt.Errorf("failed to parse payload: %w", err)
	}

	// Step 1: Check idempotency (version must be greater than applied version)
	appliedVersion, err := e.readAppliedVersion()
	if err != nil {
		return "", fmt.Errorf("failed to read applied version: %w", err)
	}

	if payload.Version <= appliedVersion {
		// Idempotent: already applied
		return fmt.Sprintf("Version %d already applied (current: %d)", payload.Version, appliedVersion), nil
	}

	// Step 2: Create staging directory
	if err := e.dirConfig.EnsureStagingDir(payload.Version); err != nil {
		return "", fmt.Errorf("failed to create staging directory: %w", err)
	}

	stagingDir := e.dirConfig.GetStagingDir(payload.Version)

	// Step 3: Render configurations
	if err := e.renderConfigurations(stagingDir, &payload); err != nil {
		e.writeLastError(payload.Version, err)
		e.dirConfig.CleanStagingDir(payload.Version)
		return "", fmt.Errorf("failed to render configurations: %w", err)
	}

	// Step 4: Execute nginx -t
	if err := e.nginxTest(stagingDir); err != nil {
		e.writeLastError(payload.Version, err)
		e.dirConfig.CleanStagingDir(payload.Version)
		return "", fmt.Errorf("nginx test failed: %w", err)
	}

	// Step 5: Atomic switch (rename staging -> live)
	liveDir := e.dirConfig.GetLiveDir()
	if err := e.atomicSwitch(stagingDir, liveDir); err != nil {
		e.writeLastError(payload.Version, err)
		return "", fmt.Errorf("failed to switch to new configuration: %w", err)
	}

	// Step 6: Update metadata
	if err := e.writeAppliedVersion(payload.Version); err != nil {
		return "", fmt.Errorf("failed to write applied version: %w", err)
	}

	if err := e.writeLastSuccessVersion(payload.Version); err != nil {
		return "", fmt.Errorf("failed to write last success version: %w", err)
	}

	// Step 7: Reload nginx
	if err := e.nginxReload(); err != nil {
		// Reload failed, but configuration is already applied
		// Log error but don't fail the task
		return fmt.Sprintf("Configuration applied (version %d), but reload failed: %v", payload.Version, err), nil
	}

	return fmt.Sprintf("Configuration applied successfully (version %d)", payload.Version), nil
}

// renderConfigurations renders all configurations to staging directory
func (e *ApplyConfigExecutor) renderConfigurations(stagingDir string, payload *ApplyConfigPayload) error {
	for _, website := range payload.Websites {
		// Render upstream (only if not redirect mode)
		if website.Origin.Mode != "redirect" {
			upstreamData := &render.UpstreamData{
				WebsiteID:    website.WebsiteID,
				UpstreamName: website.Origin.UpstreamName,
				Addresses:    make([]render.AddressData, 0, len(website.Origin.Addresses)),
			}

			for _, addr := range website.Origin.Addresses {
				upstreamData.Addresses = append(upstreamData.Addresses, render.AddressData{
					Role:     addr.Role,
					Protocol: addr.Protocol,
					Address:  addr.Address,
					Weight:   addr.Weight,
					Enabled:  addr.Enabled,
				})
			}

			if err := e.renderer.RenderUpstream(stagingDir, website.WebsiteID, upstreamData); err != nil {
				return fmt.Errorf("failed to render upstream for website %d: %w", website.WebsiteID, err)
			}
		}

		// Render server
		serverData := &render.ServerData{
			WebsiteID: website.WebsiteID,
			Domains:   make([]render.DomainData, 0, len(website.Domains)),
			Origin: render.OriginData{
				Mode:               website.Origin.Mode,
				RedirectURL:        website.Origin.RedirectURL,
				RedirectStatusCode: website.Origin.RedirectStatusCode,
				UpstreamName:       website.Origin.UpstreamName,
			},
			HTTPS: render.HTTPSData{
				Enabled:       website.HTTPS.Enabled,
				ForceRedirect: website.HTTPS.ForceRedirect,
				HSTS:          website.HTTPS.HSTS,
			},
		}

		for _, domain := range website.Domains {
			serverData.Domains = append(serverData.Domains, render.DomainData{
				Domain:    domain.Domain,
				IsPrimary: domain.IsPrimary,
			})
		}

		// Set certificate paths if HTTPS is enabled
		if website.HTTPS.Enabled && website.HTTPS.Certificate != nil {
			certID := website.HTTPS.Certificate.CertificateID
			serverData.CertPath = filepath.Join(stagingDir, "certs", fmt.Sprintf("cert_%d.pem", certID))
			serverData.KeyPath = filepath.Join(stagingDir, "certs", fmt.Sprintf("key_%d.pem", certID))

			// Write certificate files
			if err := e.renderer.WriteCertificate(stagingDir, certID, website.HTTPS.Certificate.CertPem, website.HTTPS.Certificate.KeyPem); err != nil {
				return fmt.Errorf("failed to write certificate for website %d: %w", website.WebsiteID, err)
			}
		}

		if err := e.renderer.RenderServer(stagingDir, website.WebsiteID, serverData); err != nil {
			return fmt.Errorf("failed to render server for website %d: %w", website.WebsiteID, err)
		}
	}

	return nil
}

// nginxTest executes nginx -t to validate configuration
func (e *ApplyConfigExecutor) nginxTest(stagingDir string) error {
	// For testing, we can use a mock nginx -t command
	// In production, this should be the real nginx binary

	cmd := exec.Command(e.dirConfig.NginxBin, "-t", "-c", filepath.Join(stagingDir, "nginx.conf"))
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("nginx test failed: %w, output: %s", err, string(output))
	}

	return nil
}

// atomicSwitch atomically switches staging to live
func (e *ApplyConfigExecutor) atomicSwitch(stagingDir, liveDir string) error {
	// Remove old live directory if exists
	if _, err := os.Stat(liveDir); err == nil {
		oldDir := liveDir + ".old"
		if err := os.Rename(liveDir, oldDir); err != nil {
			return fmt.Errorf("failed to backup old live directory: %w", err)
		}
		// Clean up old directory in background
		go os.RemoveAll(oldDir)
	}

	// Atomic rename
	if err := os.Rename(stagingDir, liveDir); err != nil {
		return fmt.Errorf("failed to rename staging to live: %w", err)
	}

	return nil
}

// nginxReload reloads nginx
func (e *ApplyConfigExecutor) nginxReload() error {
	cmd := exec.Command("sh", "-c", e.dirConfig.NginxReloadCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("nginx reload failed: %w, output: %s", err, string(output))
	}

	return nil
}

// readAppliedVersion reads the currently applied version
func (e *ApplyConfigExecutor) readAppliedVersion() (int64, error) {
	metaDir := e.dirConfig.GetMetaDir()
	filePath := filepath.Join(metaDir, "applied_version.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read applied version file: %w", err)
	}

	var meta VersionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0, fmt.Errorf("failed to parse applied version: %w", err)
	}

	return meta.Version, nil
}

// writeAppliedVersion writes the applied version
func (e *ApplyConfigExecutor) writeAppliedVersion(version int64) error {
	metaDir := e.dirConfig.GetMetaDir()
	filePath := filepath.Join(metaDir, "applied_version.json")

	meta := VersionMeta{
		Version:   version,
		AppliedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal applied version: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write applied version file: %w", err)
	}

	return nil
}

// writeLastSuccessVersion writes the last success version
func (e *ApplyConfigExecutor) writeLastSuccessVersion(version int64) error {
	metaDir := e.dirConfig.GetMetaDir()
	filePath := filepath.Join(metaDir, "last_success_version.json")

	meta := VersionMeta{
		Version:   version,
		AppliedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal last success version: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write last success version file: %w", err)
	}

	return nil
}

// writeLastError writes the last error
func (e *ApplyConfigExecutor) writeLastError(version int64, err error) {
	metaDir := e.dirConfig.GetMetaDir()
	filePath := filepath.Join(metaDir, "last_error.json")

	meta := ErrorMeta{
		Version: version,
		Error:   err.Error(),
		Time:    time.Now().Format(time.RFC3339),
	}

	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(filePath, data, 0644)
}
