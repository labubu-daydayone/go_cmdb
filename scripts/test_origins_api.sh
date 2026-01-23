#!/bin/bash
# T1-03 回源分组与网站回源快照 API测试脚本

BASE_URL="http://localhost:8080"
TOKEN=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数器
TOTAL=0
PASSED=0
FAILED=0

# 测试结果输出
test_result() {
    TOTAL=$((TOTAL + 1))
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        FAILED=$((FAILED + 1))
    fi
}

echo "========================================="
echo "T1-03 回源分组与网站回源快照 API测试"
echo "========================================="
echo ""

# 1. 登录获取token
echo -e "${YELLOW}[1] 登录获取token${NC}"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
    test_result 0 "登录成功，获取token"
    echo "Token: ${TOKEN:0:20}..."
else
    test_result 1 "登录失败"
    echo "Response: $LOGIN_RESP"
    exit 1
fi
echo ""

# 2. 创建origin_group（含addresses）
echo -e "${YELLOW}[2] 创建回源分组${NC}"
CREATE_GROUP_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origin-groups/create" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test-origin-group-1",
        "description": "测试回源分组",
        "addresses": [
            {
                "role": "primary",
                "protocol": "http",
                "address": "192.168.1.100:8080",
                "weight": 10,
                "enabled": true
            },
            {
                "role": "backup",
                "protocol": "https",
                "address": "192.168.1.101:8443",
                "weight": 5,
                "enabled": true
            }
        ]
    }')

GROUP_ID=$(echo $CREATE_GROUP_RESP | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$GROUP_ID" ]; then
    test_result 0 "创建回源分组成功 (ID: $GROUP_ID)"
else
    test_result 1 "创建回源分组失败"
    echo "Response: $CREATE_GROUP_RESP"
fi
echo ""

# 3. 列表origin_groups
echo -e "${YELLOW}[3] 查询回源分组列表${NC}"
LIST_GROUPS_RESP=$(curl -s -X GET "$BASE_URL/api/v1/origin-groups?page=1&pageSize=15" \
    -H "Authorization: Bearer $TOKEN")

GROUP_COUNT=$(echo $LIST_GROUPS_RESP | grep -o '"total":[0-9]*' | cut -d':' -f2)

if [ -n "$GROUP_COUNT" ] && [ "$GROUP_COUNT" -gt 0 ]; then
    test_result 0 "查询回源分组列表成功 (总数: $GROUP_COUNT)"
else
    test_result 1 "查询回源分组列表失败"
fi
echo ""

# 4. name冲突测试
echo -e "${YELLOW}[4] name冲突测试（应返回409）${NC}"
CONFLICT_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origin-groups/create" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test-origin-group-1",
        "addresses": [{"role":"primary","protocol":"http","address":"1.1.1.1:80"}]
    }')

CONFLICT_CODE=$(echo $CONFLICT_RESP | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$CONFLICT_CODE" = "3003" ]; then
    test_result 0 "name冲突检测正常 (code: 3003)"
else
    test_result 1 "name冲突检测失败 (code: $CONFLICT_CODE)"
fi
echo ""

# 5. 更新origin_group（覆盖addresses）
echo -e "${YELLOW}[5] 更新回源分组（全量覆盖addresses）${NC}"
UPDATE_GROUP_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origin-groups/update" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": $GROUP_ID,
        \"description\": \"更新后的描述\",
        \"addresses\": [
            {
                \"role\": \"primary\",
                \"protocol\": \"http\",
                \"address\": \"192.168.2.100:8080\",
                \"weight\": 20
            }
        ]
    }")

UPDATE_SUCCESS=$(echo $UPDATE_GROUP_RESP | grep -o '"code":0')

if [ -n "$UPDATE_SUCCESS" ]; then
    test_result 0 "更新回源分组成功（addresses已全量覆盖）"
else
    test_result 1 "更新回源分组失败"
fi
echo ""

# 6. 创建website（用于测试origin_set）
echo -e "${YELLOW}[6] 创建测试网站${NC}"
# 注意：这里需要先确保websites表存在，如果不存在需要手动创建
# 为了测试，我们直接使用SQL创建
mysql -h 20.2.140.226 -u root -proot123 -e "
USE go_cmdb;
INSERT INTO websites (domain, status, origin_mode, origin_group_id, origin_set_id, created_at, updated_at)
VALUES ('test-website-1.com', 'active', 'redirect', 0, 0, NOW(), NOW());
" 2>/dev/null

WEBSITE_ID=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT id FROM websites WHERE domain = 'test-website-1.com' ORDER BY id DESC LIMIT 1;
" 2>/dev/null)

if [ -n "$WEBSITE_ID" ]; then
    test_result 0 "创建测试网站成功 (ID: $WEBSITE_ID)"
else
    test_result 1 "创建测试网站失败"
fi
echo ""

# 7. 从group创建origin_set
echo -e "${YELLOW}[7] 从分组创建回源快照${NC}"
CREATE_FROM_GROUP_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/create-from-group" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"website_id\": $WEBSITE_ID,
        \"origin_group_id\": $GROUP_ID
    }")

ORIGIN_SET_ID=$(echo $CREATE_FROM_GROUP_RESP | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$ORIGIN_SET_ID" ]; then
    test_result 0 "从分组创建回源快照成功 (ID: $ORIGIN_SET_ID)"
else
    test_result 1 "从分组创建回源快照失败"
    echo "Response: $CREATE_FROM_GROUP_RESP"
fi
echo ""

# 8. 验证origin_set的source和addresses
echo -e "${YELLOW}[8] 验证回源快照数据${NC}"
ORIGIN_SET_SOURCE=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT source FROM origin_sets WHERE id = $ORIGIN_SET_ID;
" 2>/dev/null)

ORIGIN_ADDR_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT COUNT(*) FROM origin_addresses WHERE origin_set_id = $ORIGIN_SET_ID;
" 2>/dev/null)

if [ "$ORIGIN_SET_SOURCE" = "group" ] && [ "$ORIGIN_ADDR_COUNT" -gt 0 ]; then
    test_result 0 "回源快照数据正确 (source: $ORIGIN_SET_SOURCE, addresses: $ORIGIN_ADDR_COUNT)"
else
    test_result 1 "回源快照数据错误"
fi
echo ""

# 9. 创建第二个website用于manual模式
echo -e "${YELLOW}[9] 创建第二个测试网站${NC}"
mysql -h 20.2.140.226 -u root -proot123 -e "
USE go_cmdb;
INSERT INTO websites (domain, status, origin_mode, origin_group_id, origin_set_id, created_at, updated_at)
VALUES ('test-website-2.com', 'active', 'redirect', 0, 0, NOW(), NOW());
" 2>/dev/null

WEBSITE_ID_2=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT id FROM websites WHERE domain = 'test-website-2.com' ORDER BY id DESC LIMIT 1;
" 2>/dev/null)

if [ -n "$WEBSITE_ID_2" ]; then
    test_result 0 "创建第二个测试网站成功 (ID: $WEBSITE_ID_2)"
else
    test_result 1 "创建第二个测试网站失败"
fi
echo ""

# 10. 手动创建origin_set
echo -e "${YELLOW}[10] 手动创建回源快照${NC}"
CREATE_MANUAL_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/create-manual" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"website_id\": $WEBSITE_ID_2,
        \"addresses\": [
            {
                \"role\": \"primary\",
                \"protocol\": \"http\",
                \"address\": \"10.0.0.100:80\",
                \"weight\": 10
            },
            {
                \"role\": \"primary\",
                \"protocol\": \"http\",
                \"address\": \"10.0.0.101:80\",
                \"weight\": 10
            }
        ]
    }")

ORIGIN_SET_ID_2=$(echo $CREATE_MANUAL_RESP | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$ORIGIN_SET_ID_2" ]; then
    test_result 0 "手动创建回源快照成功 (ID: $ORIGIN_SET_ID_2)"
else
    test_result 1 "手动创建回源快照失败"
fi
echo ""

# 11. 更新manual origin_set
echo -e "${YELLOW}[11] 更新manual回源快照${NC}"
UPDATE_MANUAL_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/update" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"website_id\": $WEBSITE_ID_2,
        \"addresses\": [
            {
                \"role\": \"primary\",
                \"protocol\": \"https\",
                \"address\": \"10.0.0.200:443\",
                \"weight\": 15
            }
        ]
    }")

UPDATE_MANUAL_SUCCESS=$(echo $UPDATE_MANUAL_RESP | grep -o '"code":0')

if [ -n "$UPDATE_MANUAL_SUCCESS" ]; then
    test_result 0 "更新manual回源快照成功"
else
    test_result 1 "更新manual回源快照失败"
fi
echo ""

# 12. 尝试更新group origin_set（应失败）
echo -e "${YELLOW}[12] 尝试更新group回源快照（应返回409）${NC}"
UPDATE_GROUP_SET_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/update" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"website_id\": $WEBSITE_ID,
        \"addresses\": [{\"role\":\"primary\",\"protocol\":\"http\",\"address\":\"1.1.1.1:80\"}]
    }")

UPDATE_GROUP_SET_CODE=$(echo $UPDATE_GROUP_SET_RESP | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$UPDATE_GROUP_SET_CODE" = "3003" ]; then
    test_result 0 "禁止更新group回源快照（code: 3003）"
else
    test_result 1 "group回源快照更新限制失败 (code: $UPDATE_GROUP_SET_CODE)"
fi
echo ""

# 13. 删除origin_set
echo -e "${YELLOW}[13] 删除回源快照${NC}"
DELETE_ORIGIN_SET_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/delete" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"website_id\": $WEBSITE_ID_2}")

DELETE_ORIGIN_SET_SUCCESS=$(echo $DELETE_ORIGIN_SET_RESP | grep -o '"code":0')

if [ -n "$DELETE_ORIGIN_SET_SUCCESS" ]; then
    test_result 0 "删除回源快照成功"
else
    test_result 1 "删除回源快照失败"
fi
echo ""

# 14. 验证website字段变化
echo -e "${YELLOW}[14] 验证website字段变化${NC}"
WEBSITE_MODE=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT origin_mode FROM websites WHERE id = $WEBSITE_ID_2;
" 2>/dev/null)

WEBSITE_SET_ID=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT origin_set_id FROM websites WHERE id = $WEBSITE_ID_2;
" 2>/dev/null)

if [ "$WEBSITE_MODE" = "redirect" ] && [ "$WEBSITE_SET_ID" = "0" ]; then
    test_result 0 "website字段正确 (mode: $WEBSITE_MODE, set_id: $WEBSITE_SET_ID)"
else
    test_result 1 "website字段错误"
fi
echo ""

# 15. 尝试删除被引用的origin_group（应失败）
echo -e "${YELLOW}[15] 尝试删除被引用的回源分组（应返回409）${NC}"
DELETE_USED_GROUP_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origin-groups/delete" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"ids\": [$GROUP_ID]}")

DELETE_USED_GROUP_CODE=$(echo $DELETE_USED_GROUP_RESP | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$DELETE_USED_GROUP_CODE" = "3003" ]; then
    test_result 0 "禁止删除被引用的回源分组（code: 3003）"
else
    test_result 1 "回源分组删除限制失败 (code: $DELETE_USED_GROUP_CODE)"
fi
echo ""

# 16. 验证相同IP不同set权重不同
echo -e "${YELLOW}[16] 验证相同IP在不同set中可以有不同权重${NC}"
# 创建第三个website和origin_set，使用相同IP但不同权重
mysql -h 20.2.140.226 -u root -proot123 -e "
USE go_cmdb;
INSERT INTO websites (domain, status, origin_mode, origin_group_id, origin_set_id, created_at, updated_at)
VALUES ('test-website-3.com', 'active', 'redirect', 0, 0, NOW(), NOW());
" 2>/dev/null

WEBSITE_ID_3=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT id FROM websites WHERE domain = 'test-website-3.com' ORDER BY id DESC LIMIT 1;
" 2>/dev/null)

CREATE_MANUAL_3_RESP=$(curl -s -X POST "$BASE_URL/api/v1/origins/create-manual" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"website_id\": $WEBSITE_ID_3,
        \"addresses\": [
            {
                \"role\": \"primary\",
                \"protocol\": \"http\",
                \"address\": \"192.168.2.100:8080\",
                \"weight\": 50
            }
        ]
    }")

SAME_IP_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 -se "
USE go_cmdb;
SELECT COUNT(*) FROM origin_addresses WHERE address = '192.168.2.100:8080';
" 2>/dev/null)

if [ "$SAME_IP_COUNT" -ge 2 ]; then
    test_result 0 "相同IP在不同set中可以存在（count: $SAME_IP_COUNT）"
else
    test_result 1 "相同IP测试失败"
fi
echo ""

# 测试总结
echo ""
echo "========================================="
echo "测试总结"
echo "========================================="
echo -e "总计: $TOTAL"
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}✗ 部分测试失败${NC}"
    exit 1
fi
