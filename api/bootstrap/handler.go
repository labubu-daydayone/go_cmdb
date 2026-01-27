package bootstrap

import (
	"fmt"
	"net/http"

	"go_cmdb/internal/bootstrap"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles bootstrap-related requests
type Handler struct {
	db         *gorm.DB
	tokenStore *bootstrap.TokenStore
	controlURL string // Control plane URL for install script
}

// NewHandler creates a new bootstrap handler
func NewHandler(db *gorm.DB, tokenStore *bootstrap.TokenStore, controlURL string) *Handler {
	return &Handler{
		db:         db,
		tokenStore: tokenStore,
		controlURL: controlURL,
	}
}

// GetInstallScript returns the agent installation script
// GET /bootstrap/agent/install.sh?token=XXXX
func (h *Handler) GetInstallScript(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// Validate token exists (do not consume)
	exists, err := h.tokenStore.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to validate token")
		return
	}
	if !exists {
		c.String(http.StatusGone, "token not found or expired")
		return
	}

	// Generate install script
	script := h.generateInstallScript(token)
	c.Header("Content-Type", "text/x-sh")
	c.String(http.StatusOK, script)
}

func (h *Handler) generateInstallScript(token string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

TOKEN="%s"

echo "=== CDN Agent Installation Script ==="
echo "Starting installation..."

# 1. Create directories
echo "Creating directories..."
mkdir -p /etc/cdn-agent/pki
mkdir -p /var/log/cdn-agent

# 2. Download Agent binary
echo "Downloading agent binary..."
curl -fsSL http://ojbk.zip/upload/cdn_agent -o /usr/local/bin/cdn-agent
chmod +x /usr/local/bin/cdn-agent

# 3. Download PKI certificates
echo "Downloading PKI certificates..."
curl -fsSL "%s/bootstrap/agent/pki/ca.crt?token=$TOKEN" \
  -o /etc/cdn-agent/pki/ca.crt

curl -fsSL "%s/bootstrap/agent/pki/client.crt?token=$TOKEN" \
  -o /etc/cdn-agent/pki/client.crt

curl -fsSL "%s/bootstrap/agent/pki/client.key?token=$TOKEN" \
  -o /etc/cdn-agent/pki/client.key

chmod 600 /etc/cdn-agent/pki/client.key

# 4. Write config.ini
echo "Writing configuration..."
cat > /etc/cdn-agent/config.ini <<EOF
[agent]
control_plane_url = %s
listen_addr = :8080

[mtls]
ca = /etc/cdn-agent/pki/ca.crt
cert = /etc/cdn-agent/pki/client.crt
key = /etc/cdn-agent/pki/client.key
EOF

# 5. Create systemd service
echo "Creating systemd service..."
cat > /etc/systemd/system/cdn-agent.service <<EOF
[Unit]
Description=CDN Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cdn-agent --config /etc/cdn-agent/config.ini
Restart=always
RestartSec=5
StandardOutput=append:/var/log/cdn-agent/agent.log
StandardError=append:/var/log/cdn-agent/agent.log

[Install]
WantedBy=multi-user.target
EOF

# 6. Enable and start service
echo "Enabling and starting service..."
systemctl daemon-reload
systemctl enable cdn-agent
systemctl start cdn-agent

echo "=== Installation completed successfully ==="
echo "Service status:"
systemctl status cdn-agent --no-pager
`, token, h.controlURL, h.controlURL, h.controlURL, h.controlURL)
}

// GetCACert returns the CA certificate
// GET /bootstrap/agent/pki/ca.crt?token=XXXX
func (h *Handler) GetCACert(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// Validate token exists (do not consume)
	exists, err := h.tokenStore.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to validate token")
		return
	}
	if !exists {
		c.String(http.StatusGone, "token not found or expired")
		return
	}

	// TODO: Return actual CA certificate
	// For now, return a placeholder
	caCert := "-----BEGIN CERTIFICATE-----\nPlaceholder CA Certificate\n-----END CERTIFICATE-----\n"
	
	c.Header("Content-Type", "application/x-pem-file")
	c.String(http.StatusOK, caCert)
}

// GetClientCert returns the client certificate for the node
// GET /bootstrap/agent/pki/client.crt?token=XXXX
func (h *Handler) GetClientCert(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// Validate token and get node ID (do not consume)
	tokenData, err := h.tokenStore.GetTokenData(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to get token data")
		return
	}
	if tokenData == nil {
		c.String(http.StatusGone, "token not found or expired")
		return
	}

	// Get agent identity for this node
	var identity model.AgentIdentity
	if err := h.db.Where("node_id = ? AND status = ?", tokenData.NodeID, "active").
		First(&identity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.String(http.StatusNotFound, "no identity found for this node")
			return
		}
		c.String(http.StatusInternalServerError, "failed to query identity")
		return
	}

	c.Header("Content-Type", "application/x-pem-file")
	c.String(http.StatusOK, identity.CertPEM)
}

// GetClientKey returns the client private key for the node
// This is the ONLY place where token is consumed
// GET /bootstrap/agent/pki/client.key?token=XXXX
func (h *Handler) GetClientKey(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// Atomically consume token and get node ID
	nodeID, err := h.tokenStore.ConsumeToken(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusGone, "token not found, expired, or already consumed")
		return
	}

	// Get agent identity for this node
	var identity model.AgentIdentity
	if err := h.db.Where("node_id = ? AND status = ?", nodeID, "active").
		First(&identity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.String(http.StatusNotFound, "no identity found for this node")
			return
		}
		c.String(http.StatusInternalServerError, "failed to query identity")
		return
	}

	c.Header("Content-Type", "application/x-pem-file")
	c.String(http.StatusOK, identity.KeyPEM)
}
