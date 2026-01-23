#!/bin/bash

# T2-05-fix Verification Test Script
# Tests: ACME auto apply_config + retry semantics + failed challenge cleanup

set -e

BASE_URL="http://localhost:8080"
MYSQL_CMD="mysql -h127.0.0.1 -uroot -proot go_cmdb"

echo "========================================"
echo "T2-05-fix Verification Test"
echo "========================================"

# Test 1: Create ACME provider (Let's Encrypt)
echo ""
echo "[Test 1] Create ACME provider (Let's Encrypt)"
curl -X POST "$BASE_URL/api/v1/acme/providers/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "letsencrypt",
    "directoryUrl": "https://acme-v02.api.letsencrypt.org/directory",
    "requiresEab": false
  }'

# Test 2: Create ACME account
echo ""
echo "[Test 2] Create ACME account"
curl -X POST "$BASE_URL/api/v1/acme/accounts/create" \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "letsencrypt",
    "email": "admin@example.com"
  }'

# Test 3: Create domain
echo ""
echo "[Test 3] Create domain"
curl -X POST "$BASE_URL/api/v1/domains/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example.com",
    "provider": "cloudflare",
    "apiToken": "test_token",
    "zoneId": "test_zone_id"
  }'

# Test 4: Create website with ACME mode
echo ""
echo "[Test 4] Create website with ACME mode"
curl -X POST "$BASE_URL/api/v1/websites/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Example Website",
    "domains": ["www.example.com"],
    "lineGroupId": 1,
    "httpsEnabled": true,
    "certMode": "acme"
  }'

# Test 5: Create certificate request
echo ""
echo "[Test 5] Create certificate request"
CERT_REQUEST_RESP=$(curl -X POST "$BASE_URL/api/v1/acme/certificate/request" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": 1,
    "domains": ["www.example.com"],
    "websiteIds": [1]
  }')
echo "$CERT_REQUEST_RESP"
REQUEST_ID=$(echo "$CERT_REQUEST_RESP" | jq -r '.data.id')

# Test 6: Wait for ACME Worker to process (40 seconds)
echo ""
echo "[Test 6] Waiting for ACME Worker to process (40 seconds)..."
sleep 40

# Test 7: Check certificate request status
echo ""
echo "[Test 7] Check certificate request status"
curl -X GET "$BASE_URL/api/v1/acme/certificate/requests/$REQUEST_ID"

# Test 8: Verify auto-generated config_versions (SQL)
echo ""
echo "[SQL Test 1] Verify config_versions with reason=acme-issued"
$MYSQL_CMD -e "SELECT id, version, node_id, status, reason, created_at FROM config_versions WHERE reason LIKE 'acme-issued:%' ORDER BY id DESC LIMIT 5;"

# Test 9: Verify auto-generated agent_tasks (SQL)
echo ""
echo "[SQL Test 2] Verify agent_tasks with type=apply_config"
$MYSQL_CMD -e "SELECT id, node_id, type, payload, status, created_at FROM agent_tasks WHERE type='apply_config' ORDER BY id DESC LIMIT 5;"

# Test 10: Verify agent_tasks payload contains version (SQL)
echo ""
echo "[SQL Test 3] Verify agent_tasks payload contains version"
$MYSQL_CMD -e "SELECT id, payload FROM agent_tasks WHERE type='apply_config' AND payload LIKE '%version%' ORDER BY id DESC LIMIT 3;"

# Test 11: Construct a failure scenario (invalid token)
echo ""
echo "[Test 11] Create certificate request with invalid credentials (will fail)"
FAIL_REQUEST_RESP=$(curl -X POST "$BASE_URL/api/v1/acme/certificate/request" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": 1,
    "domains": ["fail.example.com"],
    "websiteIds": []
  }')
echo "$FAIL_REQUEST_RESP"
FAIL_REQUEST_ID=$(echo "$FAIL_REQUEST_RESP" | jq -r '.data.id')

# Test 12: Wait for failure (40 seconds)
echo ""
echo "[Test 12] Waiting for failure scenario (40 seconds)..."
sleep 40

# Test 13: Verify failed request status (SQL)
echo ""
echo "[SQL Test 4] Verify failed request with attempts and last_error"
$MYSQL_CMD -e "SELECT id, status, attempts, last_error, created_at FROM certificate_requests WHERE id=$FAIL_REQUEST_ID;"

# Test 14: Verify challenge TXT records cleanup (SQL)
echo ""
echo "[SQL Test 5] Verify challenge TXT records desired_state=absent after failure"
$MYSQL_CMD -e "SELECT id, domain_id, type, name, value, desired_state, owner_type, owner_id FROM domain_dns_records WHERE owner_type='acme_challenge' AND owner_id=$FAIL_REQUEST_ID ORDER BY id DESC LIMIT 5;"

# Test 15: Retry failed request
echo ""
echo "[Test 15] Retry failed request"
curl -X POST "$BASE_URL/api/v1/acme/certificate/retry" \
  -H "Content-Type: application/json" \
  -d "{\"id\": $FAIL_REQUEST_ID}"

# Test 16: Verify retry doesn't clear attempts (SQL)
echo ""
echo "[SQL Test 6] Verify retry preserves attempts"
$MYSQL_CMD -e "SELECT id, status, attempts, last_error, updated_at FROM certificate_requests WHERE id=$FAIL_REQUEST_ID;"

# Test 17: Verify certificate_bindings activated (SQL)
echo ""
echo "[SQL Test 7] Verify certificate_bindings status=active"
$MYSQL_CMD -e "SELECT id, certificate_request_id, website_id, certificate_id, status, created_at FROM certificate_bindings WHERE certificate_request_id=$REQUEST_ID ORDER BY id DESC LIMIT 5;"

# Test 18: Verify website_https.certificate_id updated (SQL)
echo ""
echo "[SQL Test 8] Verify website_https.certificate_id updated"
$MYSQL_CMD -e "SELECT id, website_id, enabled, cert_mode, certificate_id, updated_at FROM website_https WHERE website_id=1;"

# Test 19: Verify config_versions status (SQL)
echo ""
echo "[SQL Test 9] Verify config_versions status progression"
$MYSQL_CMD -e "SELECT id, version, status, reason, created_at, applied_at FROM config_versions WHERE reason LIKE 'acme-issued:%' ORDER BY id DESC LIMIT 5;"

# Test 20: Verify challenge cleanup count (SQL)
echo ""
echo "[SQL Test 10] Count challenge records by desired_state"
$MYSQL_CMD -e "SELECT desired_state, COUNT(*) as count FROM domain_dns_records WHERE owner_type='acme_challenge' GROUP BY desired_state;"

# Additional SQL Tests

# SQL Test 11: Verify certificate fingerprint uniqueness
echo ""
echo "[SQL Test 11] Verify certificate fingerprint uniqueness"
$MYSQL_CMD -e "SELECT fingerprint, COUNT(*) as count FROM certificates GROUP BY fingerprint HAVING count > 1;"

# SQL Test 12: Verify certificate_domains for SAN
echo ""
echo "[SQL Test 12] Verify certificate_domains for SAN certificates"
$MYSQL_CMD -e "SELECT cd.certificate_id, cd.domain, cd.is_wildcard, c.status FROM certificate_domains cd JOIN certificates c ON cd.certificate_id = c.id ORDER BY cd.certificate_id DESC LIMIT 10;"

# SQL Test 13: Verify attempts increment on failure
echo ""
echo "[SQL Test 13] Verify attempts increment on each failure"
$MYSQL_CMD -e "SELECT id, status, attempts, poll_max_attempts, last_error FROM certificate_requests WHERE status='failed' OR attempts > 0 ORDER BY attempts DESC LIMIT 5;"

# SQL Test 14: Verify config_versions idempotency (no duplicates)
echo ""
echo "[SQL Test 14] Verify config_versions idempotency (no duplicate reasons)"
$MYSQL_CMD -e "SELECT reason, COUNT(*) as count FROM config_versions WHERE reason LIKE 'acme-issued:%' GROUP BY reason HAVING count > 1;"

# SQL Test 15: Verify challenge cleanup completeness
echo ""
echo "[SQL Test 15] Verify all challenges for failed requests are absent"
$MYSQL_CMD -e "SELECT cr.id as request_id, cr.status, COUNT(ddr.id) as challenge_count, SUM(CASE WHEN ddr.desired_state='absent' THEN 1 ELSE 0 END) as absent_count FROM certificate_requests cr LEFT JOIN domain_dns_records ddr ON ddr.owner_type='acme_challenge' AND ddr.owner_id=cr.id WHERE cr.status='failed' GROUP BY cr.id ORDER BY cr.id DESC LIMIT 5;"

echo ""
echo "========================================"
echo "T2-05-fix Verification Test Completed"
echo "========================================"
