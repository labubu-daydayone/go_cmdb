#!/bin/bash

# T1-02 Node Groups and Line Groups API Test Script
# This script tests all CRUD operations for node groups and line groups

BASE_URL="http://localhost:8080/api/v1"
echo "=== T1-02 API Testing Script ==="
echo "Base URL: $BASE_URL"
echo ""

# Step 1: Login and get token
echo "1. Login to get token..."
TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Error: Failed to get token"
  exit 1
fi

echo "Token: $TOKEN"
echo ""

# Step 2: Create a test domain (if not exists)
echo "2. Create test domain..."
curl -s -X POST "$BASE_URL/domains/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "test-cdn.example.com",
    "description": "Test domain for T1-02"
  }' | jq '.'
echo ""

# Get domain ID
DOMAIN_ID=$(curl -s -X GET "$BASE_URL/domains?domain=test-cdn.example.com" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.data.items[0].id')

echo "Domain ID: $DOMAIN_ID"
echo ""

# Step 3: Create test node with sub IPs
echo "3. Create test node with sub IPs..."
NODE_RESPONSE=$(curl -s -X POST "$BASE_URL/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-node-for-groups",
    "mainIP": "192.168.100.1",
    "agentPort": 8080,
    "enabled": true,
    "subIPs": [
      {"ip": "192.168.100.101", "enabled": true},
      {"ip": "192.168.100.102", "enabled": true},
      {"ip": "192.168.100.103", "enabled": true}
    ]
  }')

echo "$NODE_RESPONSE" | jq '.'
echo ""

# Get sub IP IDs
SUB_IP_ID_1=$(echo "$NODE_RESPONSE" | jq -r '.data.sub_ips[0].id')
SUB_IP_ID_2=$(echo "$NODE_RESPONSE" | jq -r '.data.sub_ips[1].id')
SUB_IP_ID_3=$(echo "$NODE_RESPONSE" | jq -r '.data.sub_ips[2].id')

echo "Sub IP IDs: $SUB_IP_ID_1, $SUB_IP_ID_2, $SUB_IP_ID_3"
echo ""

# Step 4: Create node group (with subIPs)
echo "4. Create node group with sub IPs..."
NG_RESPONSE=$(curl -s -X POST "$BASE_URL/node-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-node-group-01\",
    \"description\": \"Test node group\",
    \"domainId\": $DOMAIN_ID,
    \"subIPIds\": [$SUB_IP_ID_1, $SUB_IP_ID_2]
  }")

echo "$NG_RESPONSE" | jq '.'
echo ""

NODE_GROUP_ID=$(echo "$NG_RESPONSE" | jq -r '.data.id')
NODE_GROUP_CNAME=$(echo "$NG_RESPONSE" | jq -r '.data.cname')

echo "Node Group ID: $NODE_GROUP_ID"
echo "Node Group CNAME: $NODE_GROUP_CNAME"
echo ""

# Step 5: Verify DNS A records created
echo "5. Verify DNS A records created for node group..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=node_group&ownerId=$NODE_GROUP_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 6: List node groups
echo "6. List node groups (page 1)..."
curl -s -X GET "$BASE_URL/node-groups?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 7: Create node group with name conflict
echo "7. Create node group with name conflict (expect 409)..."
curl -s -X POST "$BASE_URL/node-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-node-group-01\",
    \"domainId\": $DOMAIN_ID
  }" | jq '.'
echo ""

# Step 8: Update node group (覆盖 subIPs)
echo "8. Update node group (replace subIPs)..."
curl -s -X POST "$BASE_URL/node-groups/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": $NODE_GROUP_ID,
    \"description\": \"Updated description\",
    \"subIPIds\": [$SUB_IP_ID_2, $SUB_IP_ID_3]
  }" | jq '.'
echo ""

# Step 9: Verify old DNS records marked as error
echo "9. Verify old DNS A records marked as error..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=node_group&ownerId=$NODE_GROUP_ID&status=error" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 10: Create line group
echo "10. Create line group..."
LG_RESPONSE=$(curl -s -X POST "$BASE_URL/line-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-line-group-01\",
    \"domainId\": $DOMAIN_ID,
    \"nodeGroupId\": $NODE_GROUP_ID
  }")

echo "$LG_RESPONSE" | jq '.'
echo ""

LINE_GROUP_ID=$(echo "$LG_RESPONSE" | jq -r '.data.id')
LINE_GROUP_CNAME=$(echo "$LG_RESPONSE" | jq -r '.data.cname')

echo "Line Group ID: $LINE_GROUP_ID"
echo "Line Group CNAME: $LINE_GROUP_CNAME"
echo ""

# Step 11: Verify DNS CNAME record created
echo "11. Verify DNS CNAME record created for line group..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=line_group&ownerId=$LINE_GROUP_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 12: List line groups
echo "12. List line groups..."
curl -s -X GET "$BASE_URL/line-groups?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 13: Create another node group for switching test
echo "13. Create another node group for switching test..."
NG2_RESPONSE=$(curl -s -X POST "$BASE_URL/node-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-node-group-02\",
    \"domainId\": $DOMAIN_ID,
    \"subIPIds\": [$SUB_IP_ID_3]
  }")

NODE_GROUP_ID_2=$(echo "$NG2_RESPONSE" | jq -r '.data.id')
echo "Node Group 2 ID: $NODE_GROUP_ID_2"
echo ""

# Step 14: Update line group (switch node group)
echo "14. Update line group (switch node group)..."
curl -s -X POST "$BASE_URL/line-groups/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": $LINE_GROUP_ID,
    \"nodeGroupId\": $NODE_GROUP_ID_2
  }" | jq '.'
echo ""

# Step 15: Verify old CNAME record marked as error
echo "15. Verify old CNAME record marked as error..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=line_group&ownerId=$LINE_GROUP_ID&status=error" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 16: Delete line group
echo "16. Delete line group..."
curl -s -X POST "$BASE_URL/line-groups/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"ids\": [$LINE_GROUP_ID]
  }" | jq '.'
echo ""

# Step 17: Verify DNS records marked as error after line group deletion
echo "17. Verify DNS records marked as error after line group deletion..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=line_group&ownerId=$LINE_GROUP_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Step 18: Delete node groups
echo "18. Delete node groups..."
curl -s -X POST "$BASE_URL/node-groups/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"ids\": [$NODE_GROUP_ID, $NODE_GROUP_ID_2]
  }" | jq '.'
echo ""

# Step 19: Verify DNS records marked as error after node group deletion
echo "19. Verify DNS records marked as error after node group deletion..."
curl -s -X GET "$BASE_URL/dns-records?ownerType=node_group" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

echo "=== T1-02 API Testing Completed ==="
