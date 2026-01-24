#!/bin/bash

# T2-10-00 验收测试脚本
# API Keys（Cloudflare）管理接口测试

set -e

BASE_URL="http://20.2.140.226:8080"
MYSQL_CMD="mysql -uroot -S /data/mysql/run/mysql.sock cdn_control"

echo "=========================================="
echo "T2-10-00 验收测试开始"
echo "=========================================="

# 清理测试数据
echo ""
echo "清理测试数据..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_providers WHERE api_key_id IN (SELECT id FROM api_keys WHERE name LIKE 'test_%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domains WHERE domain LIKE 'test-domain-%';\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM api_keys WHERE name LIKE 'test_%';\""
echo "✓ 测试数据已清理"

# ==================== SQL 验证 ====================
echo ""
echo "=========================================="
echo "SQL 验证（4条）"
echo "=========================================="

# SQL验证1: 查询api_keys表结构
echo ""
echo "[SQL-1] 验证api_keys表结构..."
RESULT=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"SELECT id,name,provider,account,status FROM api_keys LIMIT 1;\" 2>&1")
if [ $? -eq 0 ]; then
    echo "✓ api_keys表结构正确"
else
    echo "✗ api_keys表结构错误"
    exit 1
fi

# SQL验证2: 验证token不被打印（检查列定义）
echo ""
echo "[SQL-2] 验证api_token字段存在但不在查询中..."
COLUMNS=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"SHOW COLUMNS FROM api_keys;\" | grep api_token")
if [ -n "$COLUMNS" ]; then
    echo "✓ api_token字段存在"
else
    echo "✗ api_token字段不存在"
    exit 1
fi

# SQL验证3: 插入测试数据并验证引用检查
echo ""
echo "[SQL-3] 插入测试数据并创建引用..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD" <<EOF
-- 插入测试API Key
INSERT INTO api_keys (name, provider, account, api_token, status, created_at, updated_at)
VALUES ('test_api_key_1', 'cloudflare', 'test@example.com', 'test_token_12345678', 'active', NOW(), NOW());

-- 获取刚插入的ID
SET @api_key_id = LAST_INSERT_ID();

-- 插入测试domain
INSERT INTO domains (domain, purpose, status, created_at, updated_at)
VALUES ('test-domain-1.com', 'unset', 'active', NOW(), NOW());

-- 获取domain ID
SET @domain_id = LAST_INSERT_ID();

-- 创建domain_dns_providers引用
INSERT INTO domain_dns_providers (domain_id, provider, provider_zone_id, api_key_id, status, created_at, updated_at)
VALUES (@domain_id, 'cloudflare', 'test_zone_123', @api_key_id, 'active', NOW(), NOW());

-- 验证引用已创建
SELECT COUNT(*) AS ref_count FROM domain_dns_providers WHERE api_key_id = @api_key_id;
EOF
echo "✓ 测试数据和引用已创建"

# SQL验证4: 验证引用计数
echo ""
echo "[SQL-4] 验证引用计数..."
REF_COUNT=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"SELECT COUNT(*) FROM domain_dns_providers WHERE api_key_id = (SELECT id FROM api_keys WHERE name = 'test_api_key_1');\" -s -N")
if [ "$REF_COUNT" -eq "1" ]; then
    echo "✓ 引用计数正确: $REF_COUNT"
else
    echo "✗ 引用计数错误: $REF_COUNT (期望: 1)"
    exit 1
fi

# ==================== curl 验证 ====================
echo ""
echo "=========================================="
echo "curl 验证（6+条）"
echo "=========================================="

# 获取JWT Token
echo ""
echo "[curl-0] 登录获取JWT Token..."
TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "✗ 登录失败，无法获取Token"
    exit 1
fi
echo "✓ Token获取成功: ${TOKEN:0:20}..."

# curl验证1: 创建API Key
echo ""
echo "[curl-1] 创建API Key..."
CREATE_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/create" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test_api_key_2",
        "provider": "cloudflare",
        "account": "test2@example.com",
        "apiToken": "test_token_abcdefgh"
    }')
CREATE_CODE=$(echo "$CREATE_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$CREATE_CODE" == "0" ]; then
    echo "✓ 创建成功"
    echo "  响应: $CREATE_RESULT"
else
    echo "✗ 创建失败"
    echo "  响应: $CREATE_RESULT"
    exit 1
fi

# curl验证2: 列表查询
echo ""
echo "[curl-2] 查询API Keys列表..."
LIST_RESULT=$(curl -s -X GET "$BASE_URL/api/v1/api-keys?page=1&pageSize=20&status=all" \
    -H "Authorization: Bearer $TOKEN")
LIST_CODE=$(echo "$LIST_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$LIST_CODE" == "0" ]; then
    echo "✓ 列表查询成功"
    # 验证返回的是masked token
    if echo "$LIST_RESULT" | grep -q '"apiTokenMasked":"\\*\\*\\*\\*'; then
        echo "✓ Token已正确masked"
    else
        echo "✗ Token未正确masked"
        exit 1
    fi
    # 验证不包含明文token
    if echo "$LIST_RESULT" | grep -q '"apiToken"'; then
        echo "✗ 响应中包含明文apiToken字段"
        exit 1
    else
        echo "✓ 响应中不包含明文apiToken"
    fi
    echo "  响应: ${LIST_RESULT:0:200}..."
else
    echo "✗ 列表查询失败"
    echo "  响应: $LIST_RESULT"
    exit 1
fi

# 获取test_api_key_2的ID
API_KEY_2_ID=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"SELECT id FROM api_keys WHERE name = 'test_api_key_2';\" -s -N")

# curl验证3: 更新name/account
echo ""
echo "[curl-3] 更新API Key的name和account..."
UPDATE_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/update" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": $API_KEY_2_ID,
        \"name\": \"test_api_key_2_updated\",
        \"account\": \"test2_updated@example.com\"
    }")
UPDATE_CODE=$(echo "$UPDATE_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$UPDATE_CODE" == "0" ]; then
    echo "✓ 更新成功"
    echo "  响应: $UPDATE_RESULT"
else
    echo "✗ 更新失败"
    echo "  响应: $UPDATE_RESULT"
    exit 1
fi

# curl验证4: 更新token（验证返回仍是masked）
echo ""
echo "[curl-4] 更新API Key的token..."
UPDATE_TOKEN_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/update" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": $API_KEY_2_ID,
        \"apiToken\": \"new_token_xyz123456\"
    }")
UPDATE_TOKEN_CODE=$(echo "$UPDATE_TOKEN_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$UPDATE_TOKEN_CODE" == "0" ]; then
    echo "✓ Token更新成功"
    # 再次查询列表验证返回的是masked token
    LIST_AFTER_UPDATE=$(curl -s -X GET "$BASE_URL/api/v1/api-keys?keyword=test_api_key_2_updated" \
        -H "Authorization: Bearer $TOKEN")
    if echo "$LIST_AFTER_UPDATE" | grep -q '"apiTokenMasked":"\\*\\*\\*\\*'; then
        echo "✓ 更新后Token仍然是masked"
    else
        echo "✗ 更新后Token未正确masked"
        exit 1
    fi
else
    echo "✗ Token更新失败"
    echo "  响应: $UPDATE_TOKEN_RESULT"
    exit 1
fi

# curl验证5: toggle-status（被引用时失败）
echo ""
echo "[curl-5] 尝试禁用被引用的API Key（应该失败）..."
API_KEY_1_ID=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"SELECT id FROM api_keys WHERE name = 'test_api_key_1';\" -s -N")
TOGGLE_FAIL_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/toggle-status" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": $API_KEY_1_ID,
        \"status\": \"inactive\"
    }")
TOGGLE_FAIL_CODE=$(echo "$TOGGLE_FAIL_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$TOGGLE_FAIL_CODE" == "3003" ]; then
    echo "✓ 依赖检查生效，禁用被引用的API Key失败（符合预期）"
    echo "  响应: $TOGGLE_FAIL_RESULT"
else
    echo "✗ 依赖检查失败，应该返回code=3003"
    echo "  响应: $TOGGLE_FAIL_RESULT"
    exit 1
fi

# curl验证6: toggle-status（未被引用时成功）
echo ""
echo "[curl-6] 禁用未被引用的API Key（应该成功）..."
TOGGLE_SUCCESS_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/toggle-status" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": $API_KEY_2_ID,
        \"status\": \"inactive\"
    }")
TOGGLE_SUCCESS_CODE=$(echo "$TOGGLE_SUCCESS_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$TOGGLE_SUCCESS_CODE" == "0" ]; then
    echo "✓ 禁用成功"
    echo "  响应: $TOGGLE_SUCCESS_RESULT"
else
    echo "✗ 禁用失败"
    echo "  响应: $TOGGLE_SUCCESS_RESULT"
    exit 1
fi

# curl验证7: delete（被引用时失败）
echo ""
echo "[curl-7] 尝试删除被引用的API Key（应该失败）..."
DELETE_FAIL_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/delete" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"ids\": [$API_KEY_1_ID]
    }")
DELETE_FAIL_CODE=$(echo "$DELETE_FAIL_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$DELETE_FAIL_CODE" == "3003" ]; then
    echo "✓ 依赖检查生效，删除被引用的API Key失败（符合预期）"
    echo "  响应: $DELETE_FAIL_RESULT"
else
    echo "✗ 依赖检查失败，应该返回code=3003"
    echo "  响应: $DELETE_FAIL_RESULT"
    exit 1
fi

# curl验证8: delete（未被引用时成功）
echo ""
echo "[curl-8] 删除未被引用的API Key（应该成功）..."
DELETE_SUCCESS_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/api-keys/delete" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"ids\": [$API_KEY_2_ID]
    }")
DELETE_SUCCESS_CODE=$(echo "$DELETE_SUCCESS_RESULT" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$DELETE_SUCCESS_CODE" == "0" ]; then
    echo "✓ 删除成功"
    echo "  响应: $DELETE_SUCCESS_RESULT"
else
    echo "✗ 删除失败"
    echo "  响应: $DELETE_SUCCESS_RESULT"
    exit 1
fi

# 清理测试数据
echo ""
echo "清理测试数据..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_providers WHERE api_key_id IN (SELECT id FROM api_keys WHERE name LIKE 'test_%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domains WHERE domain LIKE 'test-domain-%';\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM api_keys WHERE name LIKE 'test_%';\""
echo "✓ 测试数据已清理"

echo ""
echo "=========================================="
echo "T2-10-00 验收测试完成"
echo "=========================================="
echo "SQL验证: 4/4 通过"
echo "curl验证: 8/8 通过"
echo ""
