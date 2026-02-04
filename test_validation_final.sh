#!/bin/bash

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc3MDMxNzk1MywiaWF0IjoxNzcwMjMxNTUzfQ._0rwm7SNYDcImqMRwxgakFbIBZE09RpUfA2X9OnPyBk"
BASE_URL="http://20.2.140.226:8080"

echo "=========================================="
echo "Test 1: Create manual website (originSetId=null, originGroupId=null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 5,
    "domains": ["test-manual-'$(date +%s)'.example.com"],
    "origin_mode": "manual",
    "origin_addresses": [
      {
        "role": "primary",
        "protocol": "http",
        "address": "192.168.1.100:80",
        "weight": 100,
        "enabled": true
      }
    ]
  }' | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 2: Create redirect website (both null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 5,
    "domains": ["test-redirect-'$(date +%s)'.example.com"],
    "origin_mode": "redirect",
    "redirect_url": "https://example.com",
    "redirect_status_code": 301
  }' | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 3: Create group website with originGroupId only (originSetId=null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 5,
    "domains": ["test-group-'$(date +%s)'.example.com"],
    "origin_mode": "group",
    "origin_group_id": 2
  }' | python3 -m json.tool

echo ""
echo "Creating website for bind/unbind test..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/websites/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 5,
    "domains": ["test-bind-'$(date +%s)'.example.com"],
    "origin_mode": "group",
    "origin_group_id": 2
  }')

echo "$RESPONSE" | python3 -m json.tool

WEBSITE_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['item']['id'])" 2>/dev/null)

if [ -z "$WEBSITE_ID" ]; then
  echo "Failed to create website for bind test"
  exit 1
fi

echo "Website ID: ${WEBSITE_ID}"

# Get origin_set_id from the created website
ORIGIN_SET_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin)['data']['item']; print(d.get('originSetId') or '')" 2>/dev/null)

if [ -z "$ORIGIN_SET_ID" ] || [ "$ORIGIN_SET_ID" = "None" ]; then
  echo "Warning: No originSetId in response, will use origin_set_id=1 for bind test"
  ORIGIN_SET_ID=1
fi

echo "Origin Set ID: ${ORIGIN_SET_ID}"

echo ""
echo "=========================================="
echo "Test 4: Bind origin_set to website"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/origin-sets/${ORIGIN_SET_ID}/bind-website" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"website_id\": ${WEBSITE_ID}
  }" | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 5: Query website to verify originSetId is set"
echo "=========================================="
curl -s -X GET "${BASE_URL}/api/v1/websites/${WEBSITE_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool

echo ""
echo "=========================================="
echo "Validation tests completed"
echo "=========================================="
