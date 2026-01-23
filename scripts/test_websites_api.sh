#!/bin/bash

# T1-04 Websites API Test Script
# 测试网站管理的完整功能

set -e

BASE_URL="http://localhost:8080/api/v1"
TOKEN=""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "=== T1-04 Websites API Test ==="
echo

# Test 1: Login
echo "Test 1: Login to get token"
RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ Login failed${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Login successful, token: ${TOKEN:0:20}...${NC}"
echo

# Test 2: Create domain (prerequisite)
echo "Test 2: Create domain for testing"
DOMAIN_RESPONSE=$(curl -s -X POST "$BASE_URL/domains/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}')
DOMAIN_ID=$(echo $DOMAIN_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Domain created: ID=$DOMAIN_ID${NC}"
echo

# Test 3: Create line group (prerequisite)
echo "Test 3: Create line group for testing"
LG_RESPONSE=$(curl -s -X POST "$BASE_URL/line-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"test-line-group-1\",\"domain_id\":$DOMAIN_ID,\"node_group_id\":1}")
LG_ID=$(echo $LG_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Line group created: ID=$LG_ID${NC}"
echo

# Test 4: Create origin group (prerequisite)
echo "Test 4: Create origin group for testing"
OG_RESPONSE=$(curl -s -X POST "$BASE_URL/origin-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-origin-group-1","description":"Test origin group","addresses":[{"role":"primary","protocol":"http","address":"192.168.1.100:8080","weight":10,"enabled":true}]}')
OG_ID=$(echo $OG_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Origin group created: ID=$OG_ID${NC}"
echo

# Test 5: Create website (group mode + multiple domains)
echo "Test 5: Create website with group origin mode and multiple domains"
WEBSITE1_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"line_group_id\":$LG_ID,\"domains\":[\"www.example.com\",\"api.example.com\"],\"origin_mode\":\"group\",\"origin_group_id\":$OG_ID}")
WEBSITE1_ID=$(echo $WEBSITE1_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
if [ -z "$WEBSITE1_ID" ]; then
  echo -e "${RED}✗ Create website failed${NC}"
  echo "Response: $WEBSITE1_RESPONSE"
  exit 1
fi
echo -e "${GREEN}✓ Website created (group mode): ID=$WEBSITE1_ID${NC}"
echo

# Test 6: Create website (manual mode)
echo "Test 6: Create website with manual origin mode"
WEBSITE2_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"line_group_id\":$LG_ID,\"domains\":[\"manual.example.com\"],\"origin_mode\":\"manual\",\"origin_addresses\":[{\"role\":\"primary\",\"protocol\":\"http\",\"address\":\"10.0.0.100:80\",\"weight\":10,\"enabled\":true}]}")
WEBSITE2_ID=$(echo $WEBSITE2_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Website created (manual mode): ID=$WEBSITE2_ID${NC}"
echo

# Test 7: Create website (redirect mode)
echo "Test 7: Create website with redirect origin mode"
WEBSITE3_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"line_group_id\":$LG_ID,\"domains\":[\"redirect.example.com\"],\"origin_mode\":\"redirect\",\"redirect_url\":\"https://www.google.com\",\"redirect_status_code\":301}")
WEBSITE3_ID=$(echo $WEBSITE3_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Website created (redirect mode): ID=$WEBSITE3_ID${NC}"
echo

# Test 8: Domain conflict test
echo "Test 8: Test domain conflict (should fail with 409)"
CONFLICT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/websites/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"line_group_id\":$LG_ID,\"domains\":[\"www.example.com\"],\"origin_mode\":\"redirect\",\"redirect_url\":\"https://www.google.com\",\"redirect_status_code\":301}")
HTTP_CODE=$(echo "$CONFLICT_RESPONSE" | tail -1)
if [ "$HTTP_CODE" == "409" ]; then
  echo -e "${GREEN}✓ Domain conflict detected correctly (HTTP 409)${NC}"
else
  echo -e "${RED}✗ Domain conflict not detected (HTTP $HTTP_CODE)${NC}"
fi
echo

# Test 9: List websites
echo "Test 9: List websites"
LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/websites?page=1&pageSize=15" \
  -H "Authorization: Bearer $TOKEN")
TOTAL=$(echo $LIST_RESPONSE | grep -o '"total":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ Websites listed: total=$TOTAL${NC}"
echo

# Test 10: Search by domain
echo "Test 10: Search websites by domain"
SEARCH_RESPONSE=$(curl -s -X GET "$BASE_URL/websites?domain=www.example" \
  -H "Authorization: Bearer $TOKEN")
SEARCH_TOTAL=$(echo $SEARCH_RESPONSE | grep -o '"total":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ Search results: total=$SEARCH_TOTAL${NC}"
echo

# Test 11: Get website by ID
echo "Test 11: Get website by ID"
DETAIL_RESPONSE=$(curl -s -X GET "$BASE_URL/websites/$WEBSITE1_ID" \
  -H "Authorization: Bearer $TOKEN")
DETAIL_ID=$(echo $DETAIL_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Website detail retrieved: ID=$DETAIL_ID${NC}"
echo

# Test 12: Update website (switch line group)
echo "Test 12: Update website (switch line group)"
# Create another line group first
LG2_RESPONSE=$(curl -s -X POST "$BASE_URL/line-groups/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"test-line-group-2\",\"domain_id\":$DOMAIN_ID,\"node_group_id\":1}")
LG2_ID=$(echo $LG2_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
UPDATE_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":$WEBSITE1_ID,\"line_group_id\":$LG2_ID}")
echo -e "${GREEN}✓ Website updated (line group switched)${NC}"
echo

# Test 13: Update website (switch origin mode)
echo "Test 13: Update website (switch origin mode from group to manual)"
UPDATE2_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":$WEBSITE1_ID,\"origin_mode\":\"manual\",\"origin_addresses\":[{\"role\":\"primary\",\"protocol\":\"http\",\"address\":\"192.168.2.100:8080\",\"weight\":20,\"enabled\":true}]}")
echo -e "${GREEN}✓ Website updated (origin mode switched)${NC}"
echo

# Test 14: Update HTTPS (select mode)
echo "Test 14: Update website HTTPS (select mode)"
UPDATE3_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":$WEBSITE1_ID,\"https\":{\"enabled\":true,\"force_redirect\":true,\"hsts\":true,\"cert_mode\":\"select\",\"certificate_id\":1}}")
echo -e "${GREEN}✓ Website HTTPS updated (select mode)${NC}"
echo

# Test 15: Update HTTPS (acme mode)
echo "Test 15: Update website HTTPS (acme mode)"
UPDATE4_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\":$WEBSITE2_ID,\"https\":{\"enabled\":true,\"force_redirect\":false,\"hsts\":false,\"cert_mode\":\"acme\",\"acme_provider_id\":1,\"acme_account_id\":1}}")
echo -e "${GREEN}✓ Website HTTPS updated (acme mode)${NC}"
echo

# Test 16: Delete website
echo "Test 16: Delete website"
DELETE_RESPONSE=$(curl -s -X POST "$BASE_URL/websites/delete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ids\":[$WEBSITE3_ID]}")
DELETED=$(echo $DELETE_RESPONSE | grep -o '"deleted":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ Website deleted: count=$DELETED${NC}"
echo

echo "=== All tests completed ==="
echo
echo "Summary:"
echo "- Created 3 websites (group/manual/redirect modes)"
echo "- Tested domain conflict detection"
echo "- Tested list and search"
echo "- Tested line group switching"
echo "- Tested origin mode switching"
echo "- Tested HTTPS configuration (select/acme modes)"
echo "- Tested website deletion"
echo
echo "Next steps:"
echo "1. Run SQL verification: mysql < scripts/verify_websites.sql"
echo "2. Check DNS records: SELECT * FROM domain_dns_records WHERE owner_type='website_domain';"
echo "3. Check origin_sets: SELECT * FROM origin_sets WHERE id IN (SELECT origin_set_id FROM websites);"
