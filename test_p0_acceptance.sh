#!/bin/bash

BASE_URL="http://20.2.140.226:8080/api/v1"

# 获取 Token（假设已有管理员账号）
echo "=== 获取 Token ==="
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "登录失败，尝试使用默认 Token"
    # 使用之前的 Token
    TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mzg3NTYxMTMsInVzZXJfaWQiOjF9.P5Iu4LPXjwSJNKmKMGRyBDdF7zYoXqQzQxPPPxqg6BI"
fi

echo "Token: $TOKEN"
echo ""

# Test 1: 创建 manual 网站
echo "=== Test 1: 创建 manual 网站 ==="
RESP1=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "w1.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "manual"
  }')
echo "$RESP1" | python3 -m json.tool
echo ""

# 提取 website ID
WEBSITE_ID_1=$(echo $RESP1 | grep -o '"id":[0-9]*' | head -n 1 | cut -d':' -f2)
echo "Website ID 1: $WEBSITE_ID_1"
echo ""

# Test 2: 创建 group 网站
echo "=== Test 2: 创建 group 网站 ==="
RESP2=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "w2.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "group",
    "originGroupId": 2,
    "originSetId": 1
  }')
echo "$RESP2" | python3 -m json.tool
echo ""

# Test 3: 重复 domain 创建（应该失败）
echo "=== Test 3: 重复 domain 创建 ==="
RESP3=$(curl -s -X POST "$BASE_URL/websites/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "domain": "w1.4pxtech.com",
    "lineGroupId": 3,
    "originMode": "manual"
  }')
echo "$RESP3" | python3 -m json.tool
echo ""

# Test 4: 查询网站详情（验证 originGroupId 和 originSetId 为 null）
if [ -n "$WEBSITE_ID_1" ]; then
    echo "=== Test 4: 查询 manual 网站详情 ==="
    RESP4=$(curl -s -X GET "$BASE_URL/websites/$WEBSITE_ID_1" \
      -H "Authorization: Bearer $TOKEN")
    echo "$RESP4" | python3 -m json.tool
    echo ""
fi

echo "=== 验收测试完成 ==="
