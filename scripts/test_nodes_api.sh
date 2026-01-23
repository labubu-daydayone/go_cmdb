#!/bin/bash

# T1-01 Nodes API test script
# Tests all nodes CRUD operations and sub IP management

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "========================================="
echo "T1-01 Nodes API Test Suite"
echo "========================================="
echo ""

# Step 1: Get token
echo "Step 1: Getting authentication token..."
echo ""

LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "Error: Failed to get token"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

echo "Token obtained: ${TOKEN:0:20}..."
echo ""

# Step 2: Create node with subIPs
echo "========================================="
echo "Test 2: Create node with subIPs"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "node-test-01",
    "mainIP": "192.168.1.100",
    "agentPort": 8080,
    "enabled": true,
    "subIPs": [
      {"ip": "192.168.1.101", "enabled": true},
      {"ip": "192.168.1.102", "enabled": false}
    ]
  }' | jq .

echo ""

# Step 3: Create node with name conflict
echo "========================================="
echo "Test 3: Create node with name conflict"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "node-test-01",
    "mainIP": "192.168.1.200"
  }' | jq .

echo ""

# Step 4: List nodes with pagination
echo "========================================="
echo "Test 4: List nodes with pagination"
echo "========================================="
echo ""

curl -s -X GET "$API_URL/nodes?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""

# Step 5: Search by main IP
echo "========================================="
echo "Test 5: Search by main IP"
echo "========================================="
echo ""

curl -s -X GET "$API_URL/nodes?ip=192.168.1.100" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""

# Step 6: Search by sub IP
echo "========================================="
echo "Test 6: Search by sub IP"
echo "========================================="
echo ""

curl -s -X GET "$API_URL/nodes?ip=192.168.1.101" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""

# Step 7: Create another node for update test
echo "========================================="
echo "Test 7: Create another node"
echo "========================================="
echo ""

CREATE_RESPONSE=$(curl -s -X POST "$API_URL/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "node-test-02",
    "mainIP": "192.168.2.100",
    "subIPs": [
      {"ip": "192.168.2.101"},
      {"ip": "192.168.2.102"}
    ]
  }')

echo $CREATE_RESPONSE | jq .
NODE_ID=$(echo $CREATE_RESPONSE | jq -r '.data.id')
echo ""
echo "Created node ID: $NODE_ID"
echo ""

# Step 8: Update node status and enabled
echo "========================================="
echo "Test 8: Update node status and enabled"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": $NODE_ID,
    \"status\": \"online\",
    \"enabled\": false
  }" | jq .

echo ""

# Step 9: Update node with full subIPs replacement
echo "========================================="
echo "Test 9: Full subIPs replacement"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": $NODE_ID,
    \"subIPs\": [
      {\"ip\": \"192.168.2.201\", \"enabled\": true},
      {\"ip\": \"192.168.2.202\", \"enabled\": true},
      {\"ip\": \"192.168.2.203\", \"enabled\": false}
    ]
  }" | jq .

echo ""

# Step 10: Add sub IPs
echo "========================================="
echo "Test 10: Add sub IPs"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/sub-ips/add" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"nodeId\": $NODE_ID,
    \"subIPs\": [
      {\"ip\": \"192.168.2.204\", \"enabled\": true}
    ]
  }" | jq .

echo ""

# Step 11: Get node details to find sub IP ID
echo "========================================="
echo "Test 11: Get node details"
echo "========================================="
echo ""

NODE_DETAIL=$(curl -s -X GET "$API_URL/nodes?page=1&pageSize=1" \
  -H "Authorization: Bearer $TOKEN")

echo $NODE_DETAIL | jq .

SUB_IP_ID=$(echo $NODE_DETAIL | jq -r '.data.items[0].sub_ips[0].id')
echo ""
echo "Sub IP ID: $SUB_IP_ID"
echo ""

# Step 12: Toggle sub IP
echo "========================================="
echo "Test 12: Toggle sub IP enabled"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/sub-ips/toggle" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"nodeId\": $NODE_ID,
    \"subIPId\": $SUB_IP_ID,
    \"enabled\": false
  }" | jq .

echo ""

# Step 13: Delete sub IPs
echo "========================================="
echo "Test 13: Delete sub IPs"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/sub-ips/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"nodeId\": $NODE_ID,
    \"subIPIds\": [$SUB_IP_ID]
  }" | jq .

echo ""

# Step 14: Batch delete nodes
echo "========================================="
echo "Test 14: Batch delete nodes"
echo "========================================="
echo ""

curl -s -X POST "$API_URL/nodes/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"ids\": [$NODE_ID]
  }" | jq .

echo ""

echo "========================================="
echo "All tests completed!"
echo "========================================="
