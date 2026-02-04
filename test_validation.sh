#!/bin/bash

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc3MDMxNzk1MywiaWF0IjoxNzcwMjMxNTUzfQ._0rwm7SNYDcImqMRwxgakFbIBZE09RpUfA2X9OnPyBk"
BASE_URL="http://20.2.140.226:8080"

echo "=========================================="
echo "Test 1: Create manual website (originSetId=null, originGroupId=null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-manual-site",
    "lineGroupId": 1,
    "originMode": "manual",
    "origins": [
      {"address": "192.168.1.100", "port": 80, "weight": 100}
    ]
  }' | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 2: Create redirect website (both null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-redirect-site",
    "lineGroupId": 1,
    "originMode": "redirect",
    "redirectUrl": "https://example.com"
  }' | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 3: Create group website with originGroupId only (originSetId=null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/websites" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-group-site",
    "lineGroupId": 1,
    "originMode": "group",
    "originGroupId": 1
  }' | python3 -m json.tool

echo ""
echo "Saving website ID for bind test..."
WEBSITE_ID=$(curl -s -X POST "${BASE_URL}/api/v1/websites" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-bind-site",
    "lineGroupId": 1,
    "originMode": "group",
    "originGroupId": 1
  }' | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['item']['id'])")

echo "Website ID: ${WEBSITE_ID}"

echo ""
echo "=========================================="
echo "Test 4: Bind origin_set to website"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/origin-sets/1/bind" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"websiteId\": ${WEBSITE_ID}
  }" | python3 -m json.tool

echo ""
echo "=========================================="
echo "Test 5: Unbind origin_set (originSetId back to null)"
echo "=========================================="
curl -s -X POST "${BASE_URL}/api/v1/origin-sets/1/unbind" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"websiteId\": ${WEBSITE_ID}
  }" | python3 -m json.tool

echo ""
echo "=========================================="
echo "Validation tests completed"
echo "=========================================="
