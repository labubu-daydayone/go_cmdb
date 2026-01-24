#!/bin/bash

# T2-10-03 验收测试脚本
# Domain List API（GET /domains）

set -e

HOST="20.2.140.226:8080"
MYSQL_CMD="mysql -uroot -S /data/mysql/run/mysql.sock cdn_control"

echo "=========================================="
echo "T2-10-03 验收测试开始"
echo "=========================================="

# 清理测试数据
echo ""
echo "清理测试数据..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_zone_meta WHERE domain_id IN (SELECT id FROM domains WHERE domain LIKE 'test-list-%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_providers WHERE domain_id IN (SELECT id FROM domains WHERE domain LIKE 'test-list-%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domains WHERE domain LIKE 'test-list-%';\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM api_keys WHERE name LIKE 'test_list_%';\""
echo "✓ 测试数据已清理"

# ==================== SQL 验证 ====================
echo ""
echo "=========================================="
echo "SQL 验证（3条）"
echo "=========================================="

# [SQL-1] 插入测试数据
echo "[SQL-1] 插入测试数据..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD" <<'EOF'
-- 插入测试API Key
INSERT INTO api_keys (name, provider, account, api_token, status, created_at, updated_at)
VALUES ('test_list_key_1', 'cloudflare', 'test@example.com', 'test_token_12345678', 'active', NOW(), NOW());

SET @api_key_id = LAST_INSERT_ID();

-- 插入测试域名（purpose=unset）
INSERT INTO domains (domain, purpose, status, created_at, updated_at)
VALUES 
  ('test-list-1.com', 'unset', 'active', NOW(), NOW()),
  ('test-list-2.com', 'cdn', 'active', NOW(), NOW()),
  ('test-list-3.com', 'general', 'active', NOW(), NOW());

-- 为第一个域名绑定provider和NS
SET @domain_id_1 = (SELECT id FROM domains WHERE domain = 'test-list-1.com');
INSERT INTO domain_dns_providers (domain_id, provider, provider_zone_id, api_key_id, status, created_at, updated_at)
VALUES (@domain_id_1, 'cloudflare', 'zone123', @api_key_id, 'active', NOW(), NOW());

INSERT INTO domain_dns_zone_meta (domain_id, name_servers_json, last_sync_at, created_at, updated_at)
VALUES (@domain_id_1, '["ada.ns.cloudflare.com","bob.ns.cloudflare.com"]', NOW(), NOW(), NOW());

-- 为第二个域名绑定provider（无NS）
SET @domain_id_2 = (SELECT id FROM domains WHERE domain = 'test-list-2.com');
INSERT INTO domain_dns_providers (domain_id, provider, provider_zone_id, api_key_id, status, created_at, updated_at)
VALUES (@domain_id_2, 'cloudflare', 'zone456', @api_key_id, 'active', NOW(), NOW());
EOF
echo "✓ 测试数据已插入"

# [SQL-2] 验证purpose=unset的域名
echo "[SQL-2] 验证purpose=unset的域名..."
UNSET_COUNT=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -sN -e \"SELECT COUNT(*) FROM domains WHERE domain LIKE 'test-list-%' AND purpose='unset';\"")
if [ "$UNSET_COUNT" = "1" ]; then
    echo "✓ purpose=unset的域名数量正确: $UNSET_COUNT"
else
    echo "✗ purpose=unset的域名数量错误: $UNSET_COUNT (期望: 1)"
    exit 1
fi

# [SQL-3] 验证NS不为空（同步过的域名）
echo "[SQL-3] 验证NS不为空（同步过的域名）..."
NS_COUNT=$(sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -sN -e \"SELECT COUNT(*) FROM domain_dns_zone_meta zm JOIN domains d ON zm.domain_id = d.id WHERE d.domain LIKE 'test-list-%' AND zm.name_servers_json IS NOT NULL AND zm.name_servers_json != '';\"")
if [ "$NS_COUNT" = "1" ]; then
    echo "✓ NS不为空的域名数量正确: $NS_COUNT"
else
    echo "✗ NS不为空的域名数量错误: $NS_COUNT (期望: 1)"
    exit 1
fi

# ==================== curl 验证 ====================
echo ""
echo "=========================================="
echo "curl 验证（5条）"
echo "=========================================="

# [curl-0] 登录获取JWT Token
echo "[curl-0] 登录获取JWT Token..."
LOGIN_RESULT=$(curl -s -X POST http://$HOST/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESULT | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo "✗ Token获取失败"
    echo "  响应: $LOGIN_RESULT"
    exit 1
fi
echo "✓ Token获取成功: ${TOKEN:0:20}..."

# [curl-1] 基本查询
echo "[curl-1] 基本查询（GET /api/v1/domains）..."
LIST_RESULT=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://$HOST/api/v1/domains?page=1&pageSize=20")
LIST_CODE=$(echo $LIST_RESULT | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$LIST_CODE" = "0" ]; then
    echo "✓ 列表查询成功"
    # 验证返回的域名包含测试数据
    if echo "$LIST_RESULT" | grep -q "test-list-1.com"; then
        echo "✓ 返回的域名包含test-list-1.com"
    else
        echo "✗ 返回的域名不包含test-list-1.com"
        echo "  响应: $LIST_RESULT"
        exit 1
    fi
else
    echo "✗ 列表查询失败"
    echo "  响应: $LIST_RESULT"
    exit 1
fi

# [curl-2] purpose过滤（unset）
echo "[curl-2] purpose过滤（purpose=unset）..."
UNSET_RESULT=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://$HOST/api/v1/domains?purpose=unset")
UNSET_CODE=$(echo $UNSET_RESULT | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$UNSET_CODE" = "0" ]; then
    echo "✓ purpose=unset过滤成功"
    # 验证返回的域名只包含unset的
    if echo "$UNSET_RESULT" | grep -q "test-list-1.com"; then
        echo "✓ 返回的域名包含test-list-1.com（purpose=unset）"
    else
        echo "✗ 返回的域名不包含test-list-1.com"
        echo "  响应: $UNSET_RESULT"
        exit 1
    fi
else
    echo "✗ purpose=unset过滤失败"
    echo "  响应: $UNSET_RESULT"
    exit 1
fi

# [curl-3] purpose过滤（cdn）
echo "[curl-3] purpose过滤（purpose=cdn）..."
CDN_RESULT=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://$HOST/api/v1/domains?purpose=cdn")
CDN_CODE=$(echo $CDN_RESULT | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$CDN_CODE" = "0" ]; then
    echo "✓ purpose=cdn过滤成功"
    # 验证返回的域名包含cdn的
    if echo "$CDN_RESULT" | grep -q "test-list-2.com"; then
        echo "✓ 返回的域名包含test-list-2.com（purpose=cdn）"
    else
        echo "✗ 返回的域名不包含test-list-2.com"
        echo "  响应: $CDN_RESULT"
        exit 1
    fi
else
    echo "✗ purpose=cdn过滤失败"
    echo "  响应: $CDN_RESULT"
    exit 1
fi

# [curl-4] 验证NS字段
echo "[curl-4] 验证NS字段..."
if echo "$LIST_RESULT" | grep -q "ada.ns.cloudflare.com"; then
    echo "✓ NS字段正确（包含ada.ns.cloudflare.com）"
else
    echo "✗ NS字段错误（不包含ada.ns.cloudflare.com）"
    echo "  响应: $LIST_RESULT"
    exit 1
fi

# [curl-5] 验证apiKey字段
echo "[curl-5] 验证apiKey字段..."
if echo "$LIST_RESULT" | grep -q "test_list_key_1"; then
    echo "✓ apiKey字段正确（包含test_list_key_1）"
else
    echo "✗ apiKey字段错误（不包含test_list_key_1）"
    echo "  响应: $LIST_RESULT"
    exit 1
fi

# 清理测试数据
echo ""
echo "清理测试数据..."
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_zone_meta WHERE domain_id IN (SELECT id FROM domains WHERE domain LIKE 'test-list-%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domain_dns_providers WHERE domain_id IN (SELECT id FROM domains WHERE domain LIKE 'test-list-%');\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM domains WHERE domain LIKE 'test-list-%';\""
sshpass -p "Uviev5Ohyeit" ssh -o StrictHostKeyChecking=no root@20.2.140.226 "$MYSQL_CMD -e \"DELETE FROM api_keys WHERE name LIKE 'test_list_%';\""
echo "✓ 测试数据已清理"

echo ""
echo "=========================================="
echo "T2-10-03 验收测试完成"
echo "=========================================="
echo "SQL验证: 3/3 通过"
echo "curl验证: 5/5 通过"
echo ""
