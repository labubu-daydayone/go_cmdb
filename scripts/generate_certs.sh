#!/bin/bash
#
# Generate CA and certificates for mTLS communication
# Usage: ./scripts/generate_certs.sh
#

set -e

CERT_DIR="./certs"
CA_DIR="$CERT_DIR/ca"
CONTROL_DIR="$CERT_DIR/control"
AGENT_DIR="$CERT_DIR/agent"

# Create directories
mkdir -p "$CA_DIR" "$CONTROL_DIR" "$AGENT_DIR"

echo "=== Generating CA Certificate ==="

# Generate CA private key
openssl genrsa -out "$CA_DIR/ca-key.pem" 4096

# Generate CA certificate (valid for 10 years)
openssl req -new -x509 -days 3650 -key "$CA_DIR/ca-key.pem" -out "$CA_DIR/ca-cert.pem" \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=CMDB/OU=CA/CN=CMDB Root CA"

echo "✓ CA certificate generated"

echo "=== Generating Control Plane Client Certificate ==="

# Generate control plane private key
openssl genrsa -out "$CONTROL_DIR/client-key.pem" 4096

# Generate control plane CSR
openssl req -new -key "$CONTROL_DIR/client-key.pem" -out "$CONTROL_DIR/client.csr" \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=CMDB/OU=Control/CN=control-plane"

# Sign control plane certificate (valid for 1 year)
openssl x509 -req -days 365 -in "$CONTROL_DIR/client.csr" \
  -CA "$CA_DIR/ca-cert.pem" -CAkey "$CA_DIR/ca-key.pem" -CAcreateserial \
  -out "$CONTROL_DIR/client-cert.pem"

# Calculate fingerprint
CONTROL_FINGERPRINT=$(openssl x509 -in "$CONTROL_DIR/client-cert.pem" -noout -fingerprint -sha256 | cut -d= -f2)
echo "$CONTROL_FINGERPRINT" > "$CONTROL_DIR/fingerprint.txt"

echo "✓ Control plane certificate generated"
echo "  Fingerprint: $CONTROL_FINGERPRINT"

echo "=== Generating Agent Server Certificate ==="

# Generate agent private key
openssl genrsa -out "$AGENT_DIR/server-key.pem" 4096

# Generate agent CSR
openssl req -new -key "$AGENT_DIR/server-key.pem" -out "$AGENT_DIR/server.csr" \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=CMDB/OU=Agent/CN=agent-node"

# Create SAN config for agent certificate
cat > "$AGENT_DIR/san.cnf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = agent-node
IP.1 = 127.0.0.1
IP.2 = 20.2.140.226
EOF

# Sign agent certificate with SAN (valid for 1 year)
openssl x509 -req -days 365 -in "$AGENT_DIR/server.csr" \
  -CA "$CA_DIR/ca-cert.pem" -CAkey "$CA_DIR/ca-key.pem" -CAcreateserial \
  -out "$AGENT_DIR/server-cert.pem" \
  -extensions v3_req -extfile "$AGENT_DIR/san.cnf"

# Calculate fingerprint
AGENT_FINGERPRINT=$(openssl x509 -in "$AGENT_DIR/server-cert.pem" -noout -fingerprint -sha256 | cut -d= -f2)
echo "$AGENT_FINGERPRINT" > "$AGENT_DIR/fingerprint.txt"

echo "✓ Agent certificate generated"
echo "  Fingerprint: $AGENT_FINGERPRINT"

# Copy CA cert to control and agent dirs for convenience
cp "$CA_DIR/ca-cert.pem" "$CONTROL_DIR/"
cp "$CA_DIR/ca-cert.pem" "$AGENT_DIR/"

echo ""
echo "=== Certificate Generation Complete ==="
echo ""
echo "CA Certificate:      $CA_DIR/ca-cert.pem"
echo "Control Client Cert: $CONTROL_DIR/client-cert.pem"
echo "Control Client Key:  $CONTROL_DIR/client-key.pem"
echo "Agent Server Cert:   $AGENT_DIR/server-cert.pem"
echo "Agent Server Key:    $AGENT_DIR/server-key.pem"
echo ""
echo "Control Fingerprint: $CONTROL_FINGERPRINT"
echo "Agent Fingerprint:   $AGENT_FINGERPRINT"
echo ""
echo "Next steps:"
echo "1. Start agent with: AGENT_CERT=$AGENT_DIR/server-cert.pem AGENT_KEY=$AGENT_DIR/server-key.pem AGENT_CA=$AGENT_DIR/ca-cert.pem ./bin/agent"
echo "2. Start control with: CONTROL_CERT=$CONTROL_DIR/client-cert.pem CONTROL_KEY=$CONTROL_DIR/client-key.pem CONTROL_CA=$CONTROL_DIR/ca-cert.pem ./bin/cmdb"
echo "3. Create agent_identity with fingerprint: $AGENT_FINGERPRINT"
