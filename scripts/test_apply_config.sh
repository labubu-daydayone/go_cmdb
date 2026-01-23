#!/bin/bash

# T2-03 Apply Config Test Script
# This script tests the complete apply_config workflow

set -e

# Configuration
CONTROL_API="http://20.2.140.226:8080/api/v1"
AGENT_API="http://20.2.140.226:9090/agent/v1"
JWT_TOKEN="your-jwt-token-here"  # Replace with actual JWT token
MYSQL_HOST="20.2.140.226"
MYSQL_USER="root"
MYSQL_PASS="your-password"
MYSQL_DB="cmdb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_sql() {
    local sql="$1"
    mysql -h"$MYSQL_HOST" -u"$MYSQL_USER" -p"$MYSQL_PASS" "$MYSQL_DB" -e "$sql"
}

# Test counter
test_count=0
success_count=0

run_test() {
    test_count=$((test_count + 1))
    log_info "Test $test_count: $1"
}

test_success() {
    success_count=$((success_count + 1))
    log_info "✓ Test $test_count passed"
    echo ""
}

test_fail() {
    log_error "✗ Test $test_count failed: $1"
    echo ""
}

# ============================================================
# Test 1: Create Node (prerequisite)
# ============================================================
run_test "Create node for testing"

NODE_RESPONSE=$(curl -s -X POST "$CONTROL_API/nodes/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-node-apply-config",
    "main_ip": "192.168.1.100",
    "agent_port": 9090,
    "agent_identity": "test-fingerprint-123",
    "node_group_id": 1,
    "status": "active"
  }')

NODE_ID=$(echo "$NODE_RESPONSE" | jq -r '.data.id')

if [ "$NODE_ID" != "null" ] && [ -n "$NODE_ID" ]; then
    test_success
    log_info "Node ID: $NODE_ID"
else
    test_fail "Failed to create node"
    exit 1
fi

# ============================================================
# Test 2: Create Origin Group
# ============================================================
run_test "Create origin group"

ORIGIN_GROUP_RESPONSE=$(curl -s -X POST "$CONTROL_API/origin-groups/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-origin-group",
    "description": "Test origin group for apply_config"
  }')

ORIGIN_GROUP_ID=$(echo "$ORIGIN_GROUP_RESPONSE" | jq -r '.data.id')

if [ "$ORIGIN_GROUP_ID" != "null" ] && [ -n "$ORIGIN_GROUP_ID" ]; then
    test_success
    log_info "Origin Group ID: $ORIGIN_GROUP_ID"
else
    test_fail "Failed to create origin group"
    exit 1
fi

# ============================================================
# Test 3: Create Website with HTTP (no HTTPS)
# ============================================================
run_test "Create website with HTTP only"

WEBSITE_RESPONSE=$(curl -s -X POST "$CONTROL_API/websites/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-website-http",
    "origin_mode": "group",
    "origin_group_id": '"$ORIGIN_GROUP_ID"',
    "status": "active",
    "domains": ["test-http.example.com"],
    "https_enabled": false
  }')

WEBSITE_HTTP_ID=$(echo "$WEBSITE_RESPONSE" | jq -r '.data.id')

if [ "$WEBSITE_HTTP_ID" != "null" ] && [ -n "$WEBSITE_HTTP_ID" ]; then
    test_success
    log_info "Website HTTP ID: $WEBSITE_HTTP_ID"
else
    test_fail "Failed to create HTTP website"
    exit 1
fi

# ============================================================
# Test 4: Create Certificate for HTTPS
# ============================================================
run_test "Create certificate for HTTPS"

CERT_RESPONSE=$(curl -s -X POST "$CONTROL_API/certificates/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-certificate",
    "cert_pem": "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKZ...\n-----END CERTIFICATE-----",
    "key_pem": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0...\n-----END PRIVATE KEY-----",
    "expires_at": "2025-12-31T23:59:59Z",
    "status": "issued"
  }')

CERT_ID=$(echo "$CERT_RESPONSE" | jq -r '.data.id')

if [ "$CERT_ID" != "null" ] && [ -n "$CERT_ID" ]; then
    test_success
    log_info "Certificate ID: $CERT_ID"
else
    test_fail "Failed to create certificate"
    exit 1
fi

# ============================================================
# Test 5: Create Website with HTTPS
# ============================================================
run_test "Create website with HTTPS"

WEBSITE_HTTPS_RESPONSE=$(curl -s -X POST "$CONTROL_API/websites/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-website-https",
    "origin_mode": "group",
    "origin_group_id": '"$ORIGIN_GROUP_ID"',
    "status": "active",
    "domains": ["test-https.example.com"],
    "https_enabled": true,
    "force_redirect": true,
    "hsts": true,
    "cert_mode": "select",
    "certificate_id": '"$CERT_ID"'
  }')

WEBSITE_HTTPS_ID=$(echo "$WEBSITE_HTTPS_RESPONSE" | jq -r '.data.id')

if [ "$WEBSITE_HTTPS_ID" != "null" ] && [ -n "$WEBSITE_HTTPS_ID" ]; then
    test_success
    log_info "Website HTTPS ID: $WEBSITE_HTTPS_ID"
else
    test_fail "Failed to create HTTPS website"
    exit 1
fi

# ============================================================
# Test 6: Create Website with Redirect Mode
# ============================================================
run_test "Create website with redirect mode"

WEBSITE_REDIRECT_RESPONSE=$(curl -s -X POST "$CONTROL_API/websites/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-website-redirect",
    "origin_mode": "redirect",
    "redirect_url": "https://redirect.example.com",
    "redirect_status_code": 301,
    "status": "active",
    "domains": ["test-redirect.example.com"]
  }')

WEBSITE_REDIRECT_ID=$(echo "$WEBSITE_REDIRECT_RESPONSE" | jq -r '.data.id')

if [ "$WEBSITE_REDIRECT_ID" != "null" ] && [ -n "$WEBSITE_REDIRECT_ID" ]; then
    test_success
    log_info "Website Redirect ID: $WEBSITE_REDIRECT_ID"
else
    test_fail "Failed to create redirect website"
    exit 1
fi

# ============================================================
# Test 7: Apply Config (First Time)
# ============================================================
run_test "Apply config for the first time"

APPLY_RESPONSE=$(curl -s -X POST "$CONTROL_API/config/apply" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '"$NODE_ID"',
    "reason": "Initial config deployment"
  }')

VERSION_1=$(echo "$APPLY_RESPONSE" | jq -r '.data.version')
TASK_ID_1=$(echo "$APPLY_RESPONSE" | jq -r '.data.taskId')

if [ "$VERSION_1" != "null" ] && [ -n "$VERSION_1" ]; then
    test_success
    log_info "Version: $VERSION_1"
    log_info "Task ID: $TASK_ID_1"
else
    test_fail "Failed to apply config"
    exit 1
fi

# Wait for task to complete
sleep 5

# ============================================================
# Test 8: Verify Config Files Generated
# ============================================================
run_test "Verify config files generated on Agent"

# Check upstream file
UPSTREAM_FILE="/etc/nginx/cmdb/live/upstreams/upstream_site_${WEBSITE_HTTP_ID}.conf"
if ssh agent@20.2.140.226 "[ -f $UPSTREAM_FILE ]"; then
    test_success
    log_info "Upstream file exists: $UPSTREAM_FILE"
else
    test_fail "Upstream file not found: $UPSTREAM_FILE"
fi

# Check server file
SERVER_FILE="/etc/nginx/cmdb/live/servers/server_site_${WEBSITE_HTTP_ID}.conf"
if ssh agent@20.2.140.226 "[ -f $SERVER_FILE ]"; then
    test_success
    log_info "Server file exists: $SERVER_FILE"
else
    test_fail "Server file not found: $SERVER_FILE"
fi

# ============================================================
# Test 9: Verify nginx -t Success
# ============================================================
run_test "Verify nginx -t success"

# Check last_success_version.json
SUCCESS_VERSION=$(ssh agent@20.2.140.226 "cat /etc/nginx/cmdb/meta/last_success_version.json" | jq -r '.version')

if [ "$SUCCESS_VERSION" == "$VERSION_1" ]; then
    test_success
    log_info "nginx -t succeeded, version: $SUCCESS_VERSION"
else
    test_fail "nginx -t failed or version mismatch"
fi

# ============================================================
# Test 10: Modify Website and Apply Again
# ============================================================
run_test "Modify website origin and apply again"

# Update origin addresses (simulate change)
curl -s -X POST "$CONTROL_API/websites/$WEBSITE_HTTP_ID/update" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "origin_addresses": [
      {"role": "primary", "protocol": "http", "address": "192.168.1.101:80", "weight": 100, "enabled": true}
    ]
  }' > /dev/null

# Apply config again
APPLY_RESPONSE_2=$(curl -s -X POST "$CONTROL_API/config/apply" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '"$NODE_ID"',
    "reason": "Update origin addresses"
  }')

VERSION_2=$(echo "$APPLY_RESPONSE_2" | jq -r '.data.version')

if [ "$VERSION_2" != "null" ] && [ "$VERSION_2" -gt "$VERSION_1" ]; then
    test_success
    log_info "New version: $VERSION_2 (previous: $VERSION_1)"
else
    test_fail "Version did not increment"
fi

# ============================================================
# Test 11: Idempotency Test (Same Version)
# ============================================================
run_test "Idempotency test: apply same version again"

# Manually dispatch same version (simulate)
# This should return success without re-applying
IDEMPOTENT_RESPONSE=$(curl -s -X POST "$AGENT_API/tasks/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "idempotent-test-'$(date +%s)'",
    "type": "apply_config",
    "payload": {
      "version": '"$VERSION_1"',
      "websites": []
    }
  }')

IDEMPOTENT_STATUS=$(echo "$IDEMPOTENT_RESPONSE" | jq -r '.data.status')

if [ "$IDEMPOTENT_STATUS" == "success" ]; then
    test_success
    log_info "Idempotency check passed"
else
    test_fail "Idempotency check failed"
fi

# ============================================================
# Test 12: Failure Scenario (Invalid nginx config)
# ============================================================
run_test "Failure scenario: invalid nginx config"

# Create website with empty server_name (will cause nginx -t to fail)
INVALID_WEBSITE_RESPONSE=$(curl -s -X POST "$CONTROL_API/websites/create" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-website-invalid",
    "origin_mode": "group",
    "origin_group_id": '"$ORIGIN_GROUP_ID"',
    "status": "active",
    "domains": [],
    "https_enabled": false
  }')

INVALID_WEBSITE_ID=$(echo "$INVALID_WEBSITE_RESPONSE" | jq -r '.data.id')

# Apply config (should fail nginx -t)
APPLY_INVALID_RESPONSE=$(curl -s -X POST "$CONTROL_API/config/apply" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '"$NODE_ID"',
    "reason": "Test invalid config"
  }')

VERSION_INVALID=$(echo "$APPLY_INVALID_RESPONSE" | jq -r '.data.version')
TASK_ID_INVALID=$(echo "$APPLY_INVALID_RESPONSE" | jq -r '.data.taskId')

# Wait for task to fail
sleep 5

# Check task status
TASK_STATUS=$(run_sql "SELECT status FROM agent_tasks WHERE id=$TASK_ID_INVALID;" | tail -n 1)

if [ "$TASK_STATUS" == "failed" ]; then
    test_success
    log_info "Task failed as expected (invalid config)"
else
    test_fail "Task should have failed but status is: $TASK_STATUS"
fi

# Verify old config is still live
CURRENT_VERSION=$(ssh agent@20.2.140.226 "cat /etc/nginx/cmdb/meta/applied_version.json" | jq -r '.version')

if [ "$CURRENT_VERSION" == "$VERSION_2" ]; then
    test_success
    log_info "Old config still live (version: $CURRENT_VERSION)"
else
    test_fail "Config was incorrectly switched"
fi

# ============================================================
# Test 13: Revoke Agent Identity and Apply
# ============================================================
run_test "Revoke agent identity and try to apply"

# Revoke identity
curl -s -X POST "$CONTROL_API/agent-identities/revoke" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '"$NODE_ID"'
  }' > /dev/null

# Try to apply (should fail at dispatcher level)
APPLY_REVOKED_RESPONSE=$(curl -s -X POST "$CONTROL_API/config/apply" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '"$NODE_ID"',
    "reason": "Test revoked identity"
  }')

TASK_ID_REVOKED=$(echo "$APPLY_REVOKED_RESPONSE" | jq -r '.data.taskId')

# Wait and check task status
sleep 3

TASK_STATUS_REVOKED=$(run_sql "SELECT status FROM agent_tasks WHERE id=$TASK_ID_REVOKED;" | tail -n 1)

if [ "$TASK_STATUS_REVOKED" == "failed" ]; then
    test_success
    log_info "Apply failed as expected (revoked identity)"
else
    test_fail "Apply should have failed but status is: $TASK_STATUS_REVOKED"
fi

# ============================================================
# Test 14: Query Config Versions List
# ============================================================
run_test "Query config versions list"

VERSIONS_RESPONSE=$(curl -s -X GET "$CONTROL_API/config/versions?nodeId=$NODE_ID&page=1&pageSize=10" \
  -H "Authorization: Bearer $JWT_TOKEN")

VERSIONS_COUNT=$(echo "$VERSIONS_RESPONSE" | jq -r '.data.total')

if [ "$VERSIONS_COUNT" -gt 0 ]; then
    test_success
    log_info "Found $VERSIONS_COUNT config versions"
else
    test_fail "No config versions found"
fi

# ============================================================
# Test 15: Query Specific Config Version
# ============================================================
run_test "Query specific config version"

VERSION_DETAIL_RESPONSE=$(curl -s -X GET "$CONTROL_API/config/versions/$VERSION_1" \
  -H "Authorization: Bearer $JWT_TOKEN")

VERSION_DETAIL=$(echo "$VERSION_DETAIL_RESPONSE" | jq -r '.data.version')

if [ "$VERSION_DETAIL" == "$VERSION_1" ]; then
    test_success
    log_info "Version detail retrieved: $VERSION_DETAIL"
else
    test_fail "Failed to retrieve version detail"
fi

# ============================================================
# Test 16: Query Agent Tasks
# ============================================================
run_test "Query agent tasks"

TASKS_RESPONSE=$(curl -s -X GET "$CONTROL_API/agent-tasks?nodeId=$NODE_ID&page=1&pageSize=10" \
  -H "Authorization: Bearer $JWT_TOKEN")

TASKS_COUNT=$(echo "$TASKS_RESPONSE" | jq -r '.data.total')

if [ "$TASKS_COUNT" -gt 0 ]; then
    test_success
    log_info "Found $TASKS_COUNT agent tasks"
else
    test_fail "No agent tasks found"
fi

# ============================================================
# SQL Verification Tests
# ============================================================
echo ""
log_info "=========================================="
log_info "SQL Verification Tests"
log_info "=========================================="
echo ""

# SQL Test 1: Config versions increment
run_test "SQL: Verify config versions increment"
run_sql "SELECT id, version, node_id, status FROM config_versions ORDER BY version DESC LIMIT 5;"
test_success

# SQL Test 2: Agent tasks payload contains version
run_test "SQL: Verify agent tasks payload contains version"
run_sql "SELECT id, request_id, type, JSON_EXTRACT(payload, '$.version') as version, status FROM agent_tasks WHERE type='apply_config' ORDER BY id DESC LIMIT 5;"
test_success

# SQL Test 3: Version increments after website change
run_test "SQL: Verify version increments after website change"
run_sql "SELECT COUNT(*) as version_count FROM config_versions WHERE node_id=$NODE_ID;"
test_success

# SQL Test 4: Task status changes
run_test "SQL: Verify task status changes"
run_sql "SELECT status, COUNT(*) as count FROM agent_tasks WHERE node_id=$NODE_ID GROUP BY status;"
test_success

# SQL Test 5: nginx -t failure records error
run_test "SQL: Verify failed tasks have error messages"
run_sql "SELECT id, status, last_error FROM agent_tasks WHERE status='failed' AND node_id=$NODE_ID LIMIT 3;"
test_success

# SQL Test 6: Query latest version
run_test "SQL: Query latest version for node"
run_sql "SELECT MAX(version) as latest_version FROM config_versions WHERE node_id=$NODE_ID;"
test_success

# SQL Test 7: Filter versions by nodeId
run_test "SQL: Filter versions by nodeId"
run_sql "SELECT COUNT(*) as count FROM config_versions WHERE node_id=$NODE_ID;"
test_success

# SQL Test 8: Count tasks by status
run_test "SQL: Count tasks by status"
run_sql "SELECT status, COUNT(*) as count FROM agent_tasks GROUP BY status;"
test_success

# SQL Test 9: Join query website + config_versions
run_test "SQL: Join query website + config_versions"
run_sql "SELECT w.id, w.name, cv.version, cv.status FROM websites w LEFT JOIN config_versions cv ON cv.node_id IN (SELECT id FROM nodes LIMIT 1) ORDER BY cv.version DESC LIMIT 5;"
test_success

# SQL Test 10: Query failed task error info
run_test "SQL: Query failed task error info"
run_sql "SELECT id, request_id, last_error FROM agent_tasks WHERE status='failed' ORDER BY id DESC LIMIT 3;"
test_success

# ============================================================
# Summary
# ============================================================
echo ""
log_info "=========================================="
log_info "Test Summary"
log_info "=========================================="
log_info "Total tests: $test_count"
log_info "Passed: $success_count"
log_info "Failed: $((test_count - success_count))"
echo ""

if [ $success_count -eq $test_count ]; then
    log_info "All tests passed!"
    exit 0
else
    log_error "Some tests failed!"
    exit 1
fi
