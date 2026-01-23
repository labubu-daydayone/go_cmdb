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
	NginxConf      string // Default: /etc/nginx/nginx.conf
	NginxTestCmd   string // Default: nginx -t -c /etc/nginx/nginx.conf
	NginxReloadCmd string // Default: nginx -s reload
	CMDBRenderDir  string // Default: /etc/nginx/cmdb
}

// NewDirConfig creates a new directory configuration from environment variables
func NewDirConfig() *DirConfig {
	nginxConf := getEnv("NGINX_CONF", "/etc/nginx/nginx.conf")
	nginxBin := getEnv("NGINX_BIN", "nginx")
	nginxTestCmd := getEnv("NGINX_TEST_CMD", fmt.Sprintf("%s -t -c %s", nginxBin, nginxConf))

	return &DirConfig{
		NginxDir:       getEnv("NGINX_DIR", "/etc/nginx"),
		NginxBin:       nginxBin,
		NginxConf:      nginxConf,
		NginxTestCmd:   nginxTestCmd,
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

// GetVersionsDir returns the versions directory path
func (c *DirConfig) GetVersionsDir() string {
	return filepath.Join(c.CMDBRenderDir, "versions")
}

// GetVersionDir returns the directory path for a specific version
func (c *DirConfig) GetVersionDir(version int64) string {
	return filepath.Join(c.GetVersionsDir(), fmt.Sprintf("%d", version))
}

// GetLiveDir returns the live directory path (symlink to current version)
func (c *DirConfig) GetLiveDir() string {
	return filepath.Join(c.CMDBRenderDir, "live")
}

// EnsureDirectories creates all required directories
func (c *DirConfig) EnsureDirectories() error {
	dirs := []string{
		c.GetMetaDir(),
		c.GetVersionsDir(),
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

// AtomicSwitchToVersion atomically switches the live symlink to point to a specific version
func (c *DirConfig) AtomicSwitchToVersion(version int64) error {
	versionDir := c.GetVersionDir(version)
	liveDir := c.GetLiveDir()

	// Check if version directory exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version directory does not exist: %s", versionDir)
	}

	// Create temporary symlink
	tempLink := liveDir + ".tmp"
	os.Remove(tempLink) // Remove if exists

	if err := os.Symlink(versionDir, tempLink); err != nil {
		return fmt.Errorf("failed to create temp symlink: %w", err)
	}

	// Atomically rename temp symlink to live
	if err := os.Rename(tempLink, liveDir); err != nil {
		os.Remove(tempLink) // Clean up temp link
		return fmt.Errorf("failed to rename symlink: %w", err)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
