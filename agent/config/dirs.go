package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DirConfig holds directory configuration for Agent
type DirConfig struct {
	NginxDir       string // Default: /etc/nginx
	NginxBin       string // Default: nginx
	NginxReloadCmd string // Default: nginx -s reload
	CMDBRenderDir  string // Default: /etc/nginx/cmdb
}

// NewDirConfig creates a new directory configuration from environment variables
func NewDirConfig() *DirConfig {
	return &DirConfig{
		NginxDir:       getEnv("NGINX_DIR", "/etc/nginx"),
		NginxBin:       getEnv("NGINX_BIN", "nginx"),
		NginxReloadCmd: getEnv("NGINX_RELOAD_CMD", "nginx -s reload"),
		CMDBRenderDir:  getEnv("CMDB_RENDER_DIR", "/etc/nginx/cmdb"),
	}
}

// GetUpstreamsDir returns the upstreams directory path
func (c *DirConfig) GetUpstreamsDir() string {
	return filepath.Join(c.CMDBRenderDir, "upstreams")
}

// GetServersDir returns the servers directory path
func (c *DirConfig) GetServersDir() string {
	return filepath.Join(c.CMDBRenderDir, "servers")
}

// GetCertsDir returns the certs directory path
func (c *DirConfig) GetCertsDir() string {
	return filepath.Join(c.CMDBRenderDir, "certs")
}

// GetMetaDir returns the meta directory path
func (c *DirConfig) GetMetaDir() string {
	return filepath.Join(c.CMDBRenderDir, "meta")
}

// GetStagingDir returns the staging directory path for a specific version
func (c *DirConfig) GetStagingDir(version int64) string {
	return filepath.Join(c.CMDBRenderDir, ".staging", fmt.Sprintf("%d", version))
}

// GetLiveDir returns the live directory path
func (c *DirConfig) GetLiveDir() string {
	return filepath.Join(c.CMDBRenderDir, "live")
}

// EnsureDirectories creates all required directories
func (c *DirConfig) EnsureDirectories() error {
	dirs := []string{
		c.GetUpstreamsDir(),
		c.GetServersDir(),
		c.GetCertsDir(),
		c.GetMetaDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// EnsureStagingDir creates staging directory for a specific version
func (c *DirConfig) EnsureStagingDir(version int64) error {
	stagingDir := c.GetStagingDir(version)

	// Create staging root
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"upstreams", "servers", "certs"}
	for _, subdir := range subdirs {
		dir := filepath.Join(stagingDir, subdir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create staging subdirectory %s: %w", subdir, err)
		}
	}

	return nil
}

// CleanStagingDir removes staging directory for a specific version
func (c *DirConfig) CleanStagingDir(version int64) error {
	stagingDir := c.GetStagingDir(version)
	if err := os.RemoveAll(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
