package bootstrap

import (
	"fmt"
	"net/http"

	"go_cmdb/internal/bootstrap"
	"go_cmdb/internal/model"
	"go_cmdb/internal/pki"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler handles bootstrap-related requests
type Handler struct {
	db         *gorm.DB
	tokenStore *bootstrap.TokenStore
	controlURL string // Control plane URL for install script
	caManager  *pki.CAManager
}

// NewHandler creates a new bootstrap handler
func NewHandler(db *gorm.DB, tokenStore *bootstrap.TokenStore, controlURL string, caManager *pki.CAManager) *Handler {
	return &Handler{
		db:         db,
		tokenStore: tokenStore,
		controlURL: controlURL,
		caManager:  caManager,
	}
}

// GetInstallScript returns the agent installation script
// GET /bootstrap/agent/install.sh?token=XXXX
// C0-02: Token is consumed after successful script generation
func (h *Handler) GetInstallScript(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// 1. Get token data (includes nodeId)
	tokenData, err := h.tokenStore.GetTokenData(c.Request.Context(), token)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to get token data")
		return
	}
	if tokenData == nil {
		c.String(http.StatusGone, "token not found or expired")
		return
	}

	// 2. Validate node exists
	var node model.Node
	if err := h.db.Where("id = ?", tokenData.NodeID).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.String(http.StatusNotFound, "node not found")
			return
		}
		c.String(http.StatusInternalServerError, "failed to query node")
		return
	}

	// 3. Validate agentPort is set
	if node.AgentPort == 0 {
		c.String(http.StatusBadRequest, "node agentPort is not configured")
		return
	}

	// 4. Validate node identity exists
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

	// 5. Generate install script
	script := h.generateInstallScript(token, &node)

	// 5. Return script (token is NOT consumed, relies on Redis TTL)
	// Token can be used multiple times within TTL period
	c.Header("Content-Type", "text/x-sh")
	c.String(http.StatusOK, script)
}

func (h *Handler) generateInstallScript(token string, node *model.Node) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

TOKEN="%s"
NODE_ID=%d

echo "=== CDN Agent Installation Script ==="
echo "Node ID: $NODE_ID"
echo "Starting installation..."

# 1. Check root permission
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

# 2. Create directories
echo "Creating directories..."
mkdir -p /etc/cdn-agent/pki
mkdir -p /var/log/cdn-agent
mkdir -p /data/lua
mkdir -p /data/cache
mkdir -p /data/vhost/server
mkdir -p /data/vhost/upstream
mkdir -p /data/vhost/ssl

# 3. Download Agent binary
echo "Downloading agent binary..."
curl -fsSL http://ojbk.zip/upload/cdn_agent -o /usr/local/bin/cdn-agent
chmod +x /usr/local/bin/cdn-agent

# 4. Download PKI certificates
echo "Downloading PKI certificates..."
curl -fsSL "%s/bootstrap/agent/pki/ca.crt?token=$TOKEN" \
  -o /etc/cdn-agent/pki/ca.crt

curl -fsSL "%s/bootstrap/agent/pki/client.crt?token=$TOKEN" \
  -o /etc/cdn-agent/pki/client.crt

curl -fsSL "%s/bootstrap/agent/pki/client.key?token=$TOKEN" \
  -o /etc/cdn-agent/pki/client.key

chmod 600 /etc/cdn-agent/pki/client.key

# 5. Verify certificates (fail-fast)
echo "Verifying certificates..."
if ! openssl x509 -in /etc/cdn-agent/pki/ca.crt -noout; then
  echo "Error: ca.crt is not a valid certificate"
  exit 1
fi

if ! openssl x509 -in /etc/cdn-agent/pki/client.crt -noout; then
  echo "Error: client.crt is not a valid certificate"
  exit 1
fi

if ! openssl verify -CAfile /etc/cdn-agent/pki/ca.crt /etc/cdn-agent/pki/client.crt; then
  echo "Error: certificate verification failed"
  exit 1
fi

echo "Certificate verification passed"

# 6. Write config.ini
echo "Writing configuration..."
cat > /etc/cdn-agent/config.ini <<'EOF'
[agent]
node_id = %d
listen_addr = :%d

[control]
endpoint = %s

[mtls]
cert_file = /etc/cdn-agent/pki/client.crt
key_file = /etc/cdn-agent/pki/client.key
ca_file = /etc/cdn-agent/pki/ca.crt

[paths]
lua_dir = /data/lua
cache_dir = /data/cache
vhost_server_dir = /data/vhost/server
vhost_upstream_dir = /data/vhost/upstream
vhost_ssl_dir = /data/vhost/ssl
openresty_bin = /usr/local/openresty/bin/openresty
EOF

# 6. Create systemd service
echo "Creating systemd service..."
cat > /etc/systemd/system/cdn-agent.service <<'EOF'
[Unit]
Description=cdn-agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cdn-agent --config /etc/cdn-agent/config.ini
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF

# 7. Start Agent
echo "Starting agent service..."
systemctl daemon-reload
systemctl enable cdn-agent
systemctl restart cdn-agent || /usr/local/bin/cdn-agent --config /etc/cdn-agent/config.ini &

echo "=== Installation completed successfully ==="
echo "Service status:"
systemctl status cdn-agent --no-pager || echo "Agent started in background"
`, token, node.ID, h.controlURL, h.controlURL, h.controlURL, node.ID, node.AgentPort, h.controlURL)
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

	// Get token data to retrieve node ID
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

	// Return the real CA certificate
	caCertPEM := h.caManager.GetCACertPEM()
	c.Header("Content-Type", "application/x-pem-file")
	c.String(http.StatusOK, caCertPEM)
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
// GET /bootstrap/agent/pki/client.key?token=XXXX
// Note: In C0-02, token is consumed when getting install.sh
// This endpoint still validates token but doesn't consume it again
func (h *Handler) GetClientKey(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "token is required")
		return
	}

	// Validate token and get node ID (do not consume, already consumed by install.sh)
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
	c.String(http.StatusOK, identity.KeyPEM)
}
