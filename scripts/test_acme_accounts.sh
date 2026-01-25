#!/bin/bash

API_BASE="http://20.2.140.226:8080/api/v1"

# Login
echo "=== Login ==="
TOKEN=$(curl -s -X POST $API_BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')
echo "Token: ${TOKEN:0:20}..."

# Test 1: GET /api/v1/acme/providers
echo -e "\n=== Test 1: GET /api/v1/acme/providers ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/acme/providers" | jq '.'

# Test 2: GET /api/v1/acme/accounts?page=1&pageSize=10
echo -e "\n=== Test 2: GET /api/v1/acme/accounts?page=1&pageSize=10 ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/acme/accounts?page=1&pageSize=10" | jq '.'

# Test 3: GET /api/v1/acme/accounts?providerId=1
echo -e "\n=== Test 3: GET /api/v1/acme/accounts?providerId=1 ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/acme/accounts?providerId=1" | jq '.'

# Test 4: GET /api/v1/acme/accounts?status=active
echo -e "\n=== Test 4: GET /api/v1/acme/accounts?status=active ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/acme/accounts?status=active" | jq '.'

# Test 5: GET /api/v1/acme/accounts/defaults
echo -e "\n=== Test 5: GET /api/v1/acme/accounts/defaults ==="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/acme/accounts/defaults" | jq '.'

echo -e "\n=== Tests completed ==="
