#!/bin/bash
BASE_URL="http://20.2.140.226:8080/api/v1"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc3MDMyMDczNCwiaWF0IjoxNzcwMjM0MzM0fQ.fxH8_yqVYsrPjUAkOqm1CYIkNfP4YVF98fD6HD6ikF8"

echo "=== Test 1: 创建 manual 网站 ==="
RESP1=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "test-manual-'$(date +%s)'.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "manual"
  }')
echo "$RESP1" | python3 -m json.tool
WEBSITE_ID_1=$(echo $RESP1 | grep -o '"id":[0-9]*' | head -n 1 | cut -d':' -f2)
echo ""

echo "=== Test 2: 创建 group 网站 ==="
RESP2=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "test-group-'$(date +%s)'.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "group",
    "originGroupId": 2,
    "originSetId": 1
  }')
echo "$RESP2" | python3 -m json.tool
WEBSITE_ID_2=$(echo $RESP2 | grep -o '"id":[0-9]*' | head -n 1 | cut -d':' -f2)
echo ""

echo "=== Test 3: 重复 domain 创建 ==="
RESP3=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "test-manual-'$(date +%s)'.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "manual"
  }')
DOMAIN_TEST=$(echo $RESP3 | grep -o '"domain":"[^"]*"' | cut -d'"' -f4)
RESP3_DUP=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "'$DOMAIN_TEST'",
    "lineGroupId": 3,
    "originMode": "manual"
  }')
echo "$RESP3_DUP" | python3 -m json.tool
echo ""

if [ -n "$WEBSITE_ID_1" ]; then
    echo "=== Test 4: 查询 manual 网站详情 (ID: $WEBSITE_ID_1) ==="
    RESP4=$(curl -s -X GET "$BASE_URL/websites/$WEBSITE_ID_1" \
      -H "Authorization: Bearer $TOKEN")
    echo "$RESP4" | python3 -m json.tool
    echo ""
fi

if [ -n "$WEBSITE_ID_2" ]; then
    echo "=== Test 5: 查询 group 网站详情 (ID: $WEBSITE_ID_2) ==="
    RESP5=$(curl -s -X GET "$BASE_URL/websites/$WEBSITE_ID_2" \
      -H "Authorization: Bearer $TOKEN")
    echo "$RESP5" | python3 -m json.tool
    echo ""
fi

echo "=== 验收测试完成 ==="
