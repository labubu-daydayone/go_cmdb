#!/bin/bash

# B0-01-02: 发布任务创建 - 验收测试脚本
# 包含10+条SQL验证和6+条curl验证

set -e

echo "========================================="
echo "B0-01-02 Release Create - Acceptance Test"
echo "========================================="
echo ""

# 数据库连接信息（根据实际环境修改）
DB_HOST="${DB_HOST:-20.2.140.226}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-root123}"
DB_NAME="${DB_NAME:-cmdb}"
API_BASE="${API_BASE:-http://localhost:8080}"

echo "Database: $DB_HOST/$DB_NAME"
echo "API Base: $API_BASE"
echo ""

# ============================================
# SQL验证（12条）
# ============================================

echo "=== SQL Validation ==="
echo ""

# 清理测试数据
echo "[SQL-00] Cleanup test data"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
DELETE FROM release_task_nodes WHERE release_task_id IN (SELECT id FROM release_tasks WHERE version >= 1000);
DELETE FROM release_tasks WHERE version >= 1000;
DELETE FROM nodes WHERE id >= 9000;
" 2>/dev/null
echo "  ✓ Test data cleaned"
echo ""

# 1. 插入3个online节点
echo "[SQL-01] Insert 3 online nodes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO nodes (id, name, ip, status, enabled, created_at, updated_at)
VALUES 
(9001, 'test-node-1', '192.168.1.1', 'online', 1, NOW(), NOW()),
(9002, 'test-node-2', '192.168.1.2', 'online', 1, NOW(), NOW()),
(9003, 'test-node-3', '192.168.1.3', 'online', 1, NOW(), NOW());
" 2>/dev/null
echo "  ✓ 3 online nodes inserted"
echo ""

# 2. 插入1个offline节点
echo "[SQL-02] Insert 1 offline node"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO nodes (id, name, ip, status, enabled, created_at, updated_at)
VALUES (9004, 'test-node-4', '192.168.1.4', 'offline', 1, NOW(), NOW());
" 2>/dev/null
echo "  ✓ 1 offline node inserted"
echo ""

# 3. 插入1个disabled节点
echo "[SQL-03] Insert 1 disabled node"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO nodes (id, name, ip, status, enabled, created_at, updated_at)
VALUES (9005, 'test-node-5', '192.168.1.5', 'online', 0, NOW(), NOW());
" 2>/dev/null
echo "  ✓ 1 disabled node inserted"
echo ""

# 4. 验证节点插入
echo "[SQL-04] Verify nodes inserted"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT id, name, status, enabled FROM nodes WHERE id >= 9000 ORDER BY id;
" 2>/dev/null
echo ""

# 5. 查询在线节点（应返回3个）
echo "[SQL-05] Query online nodes (should return 3)"
ONLINE_COUNT=$(mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -se "
SELECT COUNT(*) FROM nodes WHERE enabled=1 AND status='online' AND id >= 9000;
" 2>/dev/null)
echo "  Online nodes count: $ONLINE_COUNT"
if [ "$ONLINE_COUNT" -eq 3 ]; then
    echo "  ✓ Online nodes count correct"
else
    echo "  ✗ Online nodes count incorrect (expected 3, got $ONLINE_COUNT)"
fi
echo ""

# ============================================
# curl验证（8条）
# ============================================

echo "=== curl Validation ==="
echo ""

# 0. 登录获取JWT token
echo "[CURL-00] Login to get JWT token"
LOGIN_RESP=$(curl -s -X POST "$API_BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo "  ✗ Login failed, cannot get token"
    echo "  Response: $LOGIN_RESP"
    exit 1
fi
echo "  ✓ Token: ${TOKEN:0:20}..."
echo ""

# 1. 无token → 401
echo "[CURL-01] No token → 401"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -d '{"target":"cdn"}')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | head -n-1)
echo "  HTTP Code: $HTTP_CODE"
echo "  Response: $BODY"
if [ "$HTTP_CODE" -eq 401 ]; then
    echo "  ✓ 401 returned correctly"
else
    echo "  ✗ Expected 401, got $HTTP_CODE"
fi
echo ""

# 2. 3个online node → batch1=1个，batch2=2个
echo "[CURL-02] 3 online nodes → batch1=1, batch2=2"
RESP=$(curl -s -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"cdn","reason":"test 3 nodes"}')
echo "  Response: $RESP"
RELEASE_ID_1=$(echo $RESP | grep -o '"releaseId":[0-9]*' | cut -d':' -f2)
VERSION_1=$(echo $RESP | grep -o '"version":[0-9]*' | cut -d':' -f2)
TOTAL_NODES=$(echo $RESP | grep -o '"totalNodes":[0-9]*' | cut -d':' -f2)
echo "  Release ID: $RELEASE_ID_1"
echo "  Version: $VERSION_1"
echo "  Total Nodes: $TOTAL_NODES"
if [ "$TOTAL_NODES" -eq 3 ]; then
    echo "  ✓ Total nodes correct (3)"
else
    echo "  ✗ Total nodes incorrect (expected 3, got $TOTAL_NODES)"
fi
echo ""

# 3. 验证batch分组（SQL）
echo "[SQL-06] Verify batch allocation"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT batch, COUNT(*) as node_count, GROUP_CONCAT(node_id ORDER BY node_id) as node_ids
FROM release_task_nodes
WHERE release_task_id=$RELEASE_ID_1
GROUP BY batch
ORDER BY batch;
" 2>/dev/null
echo ""

# 4. 验证batch=1只有1个节点
echo "[SQL-07] Verify batch=1 has 1 node"
BATCH1_COUNT=$(mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -se "
SELECT COUNT(*) FROM release_task_nodes WHERE release_task_id=$RELEASE_ID_1 AND batch=1;
" 2>/dev/null)
echo "  Batch 1 node count: $BATCH1_COUNT"
if [ "$BATCH1_COUNT" -eq 1 ]; then
    echo "  ✓ Batch 1 has 1 node"
else
    echo "  ✗ Batch 1 incorrect (expected 1, got $BATCH1_COUNT)"
fi
echo ""

# 5. 验证batch=2有2个节点
echo "[SQL-08] Verify batch=2 has 2 nodes"
BATCH2_COUNT=$(mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -se "
SELECT COUNT(*) FROM release_task_nodes WHERE release_task_id=$RELEASE_ID_1 AND batch=2;
" 2>/dev/null)
echo "  Batch 2 node count: $BATCH2_COUNT"
if [ "$BATCH2_COUNT" -eq 2 ]; then
    echo "  ✓ Batch 2 has 2 nodes"
else
    echo "  ✗ Batch 2 incorrect (expected 2, got $BATCH2_COUNT)"
fi
echo ""

# 6. 重复调用两次 → version不同，releaseId不同
echo "[CURL-03] Repeat call → different version and releaseId"
RESP=$(curl -s -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"cdn","reason":"test repeat call"}')
echo "  Response: $RESP"
RELEASE_ID_2=$(echo $RESP | grep -o '"releaseId":[0-9]*' | cut -d':' -f2)
VERSION_2=$(echo $RESP | grep -o '"version":[0-9]*' | cut -d':' -f2)
echo "  Release ID 1: $RELEASE_ID_1, Release ID 2: $RELEASE_ID_2"
echo "  Version 1: $VERSION_1, Version 2: $VERSION_2"
if [ "$RELEASE_ID_1" != "$RELEASE_ID_2" ] && [ "$VERSION_1" != "$VERSION_2" ]; then
    echo "  ✓ Release ID and version are different"
else
    echo "  ✗ Release ID or version are same"
fi
echo ""

# 7. 禁用所有节点 → 0个online node → 409 + 3003
echo "[SQL-09] Disable all online nodes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
UPDATE nodes SET enabled=0 WHERE id >= 9000 AND id <= 9003;
" 2>/dev/null
echo "  ✓ All online nodes disabled"
echo ""

echo "[CURL-04] 0 online nodes → 409 + code=3003"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"cdn"}')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | head -n-1)
echo "  HTTP Code: $HTTP_CODE"
echo "  Response: $BODY"
CODE=$(echo $BODY | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$HTTP_CODE" -eq 409 ] && [ "$CODE" -eq 3003 ]; then
    echo "  ✓ 409 + code=3003 returned correctly"
else
    echo "  ✗ Expected 409 + code=3003, got $HTTP_CODE + code=$CODE"
fi
echo ""

# 8. 恢复1个节点 → 1个online node → 只生成batch1
echo "[SQL-10] Enable 1 node"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
UPDATE nodes SET enabled=1 WHERE id=9001;
" 2>/dev/null
echo "  ✓ 1 node enabled"
echo ""

echo "[CURL-05] 1 online node → only batch1"
RESP=$(curl -s -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"cdn","reason":"test 1 node"}')
echo "  Response: $RESP"
RELEASE_ID_3=$(echo $RESP | grep -o '"releaseId":[0-9]*' | cut -d':' -f2)
TOTAL_NODES=$(echo $RESP | grep -o '"totalNodes":[0-9]*' | cut -d':' -f2)
BATCH_COUNT=$(echo $RESP | grep -o '"batch":[0-9]*' | wc -l)
echo "  Release ID: $RELEASE_ID_3"
echo "  Total Nodes: $TOTAL_NODES"
echo "  Batch Count: $BATCH_COUNT"
if [ "$TOTAL_NODES" -eq 1 ] && [ "$BATCH_COUNT" -eq 1 ]; then
    echo "  ✓ Only batch1 generated"
else
    echo "  ✗ Expected 1 node and 1 batch, got $TOTAL_NODES nodes and $BATCH_COUNT batches"
fi
echo ""

# 9. 验证release_tasks表
echo "[SQL-11] Verify release_tasks"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT id, type, target, version, status, total_nodes, success_nodes, failed_nodes
FROM release_tasks
WHERE id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3)
ORDER BY id;
" 2>/dev/null
echo ""

# 10. 验证release_task_nodes表
echo "[SQL-12] Verify release_task_nodes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT release_task_id, node_id, batch, status
FROM release_task_nodes
WHERE release_task_id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3)
ORDER BY release_task_id, batch, node_id;
" 2>/dev/null
echo ""

# 11. 测试invalid target
echo "[CURL-06] Invalid target → 400 + code=2002"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"invalid"}')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | head -n-1)
echo "  HTTP Code: $HTTP_CODE"
echo "  Response: $BODY"
CODE=$(echo $BODY | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$HTTP_CODE" -eq 400 ] && [ "$CODE" -eq 2002 ]; then
    echo "  ✓ 400 + code=2002 returned correctly"
else
    echo "  ✗ Expected 400 + code=2002, got $HTTP_CODE + code=$CODE"
fi
echo ""

# 12. 测试missing target
echo "[CURL-07] Missing target → 400 + code=2002"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/api/v1/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}')
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | head -n-1)
echo "  HTTP Code: $HTTP_CODE"
echo "  Response: $BODY"
CODE=$(echo $BODY | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$HTTP_CODE" -eq 400 ] && [ "$CODE" -eq 2002 ]; then
    echo "  ✓ 400 + code=2002 returned correctly"
else
    echo "  ✗ Expected 400 + code=2002, got $HTTP_CODE + code=$CODE"
fi
echo ""

# 13. 验证version唯一性
echo "[SQL-13] Verify version uniqueness"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT version, COUNT(*) as count
FROM release_tasks
WHERE id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3)
GROUP BY version
HAVING count > 1;
" 2>/dev/null
DUPLICATE_COUNT=$(mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -se "
SELECT COUNT(*) FROM (
  SELECT version, COUNT(*) as count
  FROM release_tasks
  WHERE id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3)
  GROUP BY version
  HAVING count > 1
) AS duplicates;
" 2>/dev/null)
if [ "$DUPLICATE_COUNT" -eq 0 ]; then
    echo "  ✓ No duplicate versions"
else
    echo "  ✗ Found $DUPLICATE_COUNT duplicate versions"
fi
echo ""

# 清理测试数据
echo "[SQL-14] Cleanup test data"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
DELETE FROM release_task_nodes WHERE release_task_id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3);
DELETE FROM release_tasks WHERE id IN ($RELEASE_ID_1, $RELEASE_ID_2, $RELEASE_ID_3);
DELETE FROM nodes WHERE id >= 9000;
" 2>/dev/null
echo "  ✓ Test data cleaned"
echo ""

echo "=== SQL Validation Completed ==="
echo ""

# ============================================
# Go Test
# ============================================

echo "=== Go Test ==="
echo ""

echo "[GO-TEST] Running go test ./..."
cd /home/ubuntu/go_cmdb_new
go test ./... 2>&1 | grep -E "(PASS|FAIL|ok|FAIL)" | head -20
echo ""

echo "=== Go Test Completed ==="
echo ""

echo "========================================="
echo "B0-01-02 Acceptance Test Completed"
echo "========================================="
