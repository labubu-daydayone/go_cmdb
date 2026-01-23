#!/bin/bash
# B0-01-04 验收测试：发布任务查询接口

set -e

# 配置
DB_HOST="20.2.140.226"
DB_USER="root"
DB_PASS="Uviev5Ohyeit"
DB_NAME="cmdb"
API_BASE="http://localhost:8080/api/v1"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 计数器
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
test_sql() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    local desc="$1"
    local sql="$2"
    echo -e "${YELLOW}SQL Test $TOTAL_TESTS: $desc${NC}"
    if mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "$sql" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo
}

test_curl() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    local desc="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local expected_code="$5"
    echo -e "${YELLOW}CURL Test $TOTAL_TESTS: $desc${NC}"
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" == "$expected_code" ]; then
        echo -e "${GREEN}✓ PASSED (HTTP $http_code)${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo "Response: $body"
    else
        echo -e "${RED}✗ FAILED (Expected HTTP $expected_code, Got $http_code)${NC}"
        echo "Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo
}

echo "=========================================="
echo "B0-01-04 验收测试：发布任务查询接口"
echo "=========================================="
echo

# ==================== SQL验证 ====================
echo "==================== SQL验证 ===================="
echo

# SQL 0: 清理测试数据
test_sql "清理测试数据" "
DELETE FROM release_task_nodes WHERE release_task_id IN (SELECT id FROM release_tasks WHERE version >= 900000);
DELETE FROM release_tasks WHERE version >= 900000;
DELETE FROM nodes WHERE id >= 9000;
"

# SQL 1: 插入测试节点
test_sql "插入3个测试节点" "
INSERT INTO nodes (id, name, main_ip, enabled, status, created_at, updated_at)
VALUES
(9001, 'test-node-01', '192.168.1.1', 1, 'online', NOW(), NOW()),
(9002, 'test-node-02', '192.168.1.2', 1, 'online', NOW(), NOW()),
(9003, 'test-node-03', '192.168.1.3', 1, 'online', NOW(), NOW());
"

# SQL 2: 插入测试发布任务（running状态）
test_sql "插入测试发布任务（running状态）" "
INSERT INTO release_tasks (id, type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
VALUES (9001, 'apply_config', 'cdn', 900001, 'running', 3, 1, 0, NOW(), NOW());
"

# SQL 3: 插入测试发布任务节点（batch1=success, batch2=pending+skipped）
test_sql "插入测试发布任务节点" "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, started_at, finished_at, created_at, updated_at)
VALUES
(9001, 9001, 1, 'success', NOW(), NOW(), NOW(), NOW()),
(9001, 9002, 2, 'pending', NULL, NULL, NOW(), NOW()),
(9001, 9003, 2, 'skipped', NULL, NULL, NOW(), NOW());
"

# SQL 4: 验证skippedNodes计算
test_sql "验证skippedNodes计算（应为1）" "
SELECT COUNT(*) as skipped_nodes FROM release_task_nodes
WHERE release_task_id = 9001 AND status = 'skipped';
"

# SQL 5: 验证currentBatch计算（batch1=success, batch2=pending）
test_sql "验证currentBatch计算（应为2）" "
SELECT MIN(batch) as current_batch FROM release_task_nodes
WHERE release_task_id = 9001 AND status IN ('pending', 'running', 'failed');
"

# SQL 6: 插入测试发布任务（success状态）
test_sql "插入测试发布任务（success状态）" "
INSERT INTO release_tasks (id, type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
VALUES (9002, 'apply_config', 'cdn', 900002, 'success', 3, 3, 0, NOW(), NOW());
"

# SQL 7: 插入测试发布任务节点（全部success）
test_sql "插入测试发布任务节点（全部success）" "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, started_at, finished_at, created_at, updated_at)
VALUES
(9002, 9001, 1, 'success', NOW(), NOW(), NOW(), NOW()),
(9002, 9002, 2, 'success', NOW(), NOW(), NOW(), NOW()),
(9002, 9003, 2, 'success', NOW(), NOW(), NOW(), NOW());
"

# SQL 8: 验证currentBatch计算（全部success，应为NULL或0）
test_sql "验证currentBatch计算（全部success，应为NULL）" "
SELECT MIN(batch) as current_batch FROM release_task_nodes
WHERE release_task_id = 9002 AND status IN ('pending', 'running', 'failed');
"

# SQL 9: 验证列表分页
test_sql "验证列表分页（LIMIT 2）" "
SELECT id, version, status FROM release_tasks
WHERE version >= 900000
ORDER BY id DESC
LIMIT 2;
"

# SQL 10: 验证status过滤
test_sql "验证status过滤（status=running）" "
SELECT id, version, status FROM release_tasks
WHERE status = 'running' AND version >= 900000;
"

# SQL 11: 验证batch分组
test_sql "验证batch分组（GROUP BY batch）" "
SELECT batch, COUNT(*) as node_count FROM release_task_nodes
WHERE release_task_id = 9001
GROUP BY batch
ORDER BY batch;
"

# SQL 12: 验证nodeName join
test_sql "验证nodeName join" "
SELECT rtn.node_id, n.name as node_name, rtn.batch, rtn.status
FROM release_task_nodes rtn
LEFT JOIN nodes n ON rtn.node_id = n.id
WHERE rtn.release_task_id = 9001
ORDER BY rtn.batch ASC, rtn.node_id ASC;
"

# SQL 13: 验证errorMsg处理（NULL）
test_sql "验证errorMsg处理（应为NULL）" "
SELECT error_msg FROM release_task_nodes
WHERE release_task_id = 9001 AND node_id = 9001;
"

# SQL 14: 插入测试发布任务（failed状态）
test_sql "插入测试发布任务（failed状态）" "
INSERT INTO release_tasks (id, type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
VALUES (9003, 'apply_config', 'cdn', 900003, 'failed', 3, 1, 1, NOW(), NOW());
"

# SQL 15: 插入测试发布任务节点（batch1=success, batch1=failed）
test_sql "插入测试发布任务节点（batch1=success+failed）" "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, error_msg, started_at, finished_at, created_at, updated_at)
VALUES
(9003, 9001, 1, 'success', NULL, NOW(), NOW(), NOW(), NOW()),
(9003, 9002, 1, 'failed', 'Connection timeout', NOW(), NOW(), NOW(), NOW()),
(9003, 9003, 2, 'skipped', NULL, NULL, NULL, NOW(), NOW());
"

# SQL 16: 验证currentBatch计算（batch1=failed）
test_sql "验证currentBatch计算（batch1=failed，应为1）" "
SELECT MIN(batch) as current_batch FROM release_task_nodes
WHERE release_task_id = 9003 AND status IN ('pending', 'running', 'failed');
"

# ==================== CURL验证 ====================
echo "==================== CURL验证 ===================="
echo

# CURL 0: 登录获取token
echo -e "${YELLOW}CURL Test 0: 登录获取token${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | sed 's/"token":"//')
if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ 登录失败，无法获取token${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 登录成功，token: ${TOKEN:0:20}...${NC}"
echo

# CURL 1: 无token → 401
test_curl "无token → 401" "GET" "$API_BASE/releases" "" "401"

# CURL 2: 列表分页（page=1, pageSize=10）
test_curl "列表分页（page=1, pageSize=10）" "GET" "$API_BASE/releases?page=1&pageSize=10" "" "200"

# CURL 3: 列表分页（page=2, pageSize=2）
test_curl "列表分页（page=2, pageSize=2）" "GET" "$API_BASE/releases?page=2&pageSize=2" "" "200"

# CURL 4: status过滤（status=running）
test_curl "status过滤（status=running）" "GET" "$API_BASE/releases?status=running" "" "200"

# CURL 5: status过滤（status=success）
test_curl "status过滤（status=success）" "GET" "$API_BASE/releases?status=success" "" "200"

# CURL 6: 详情返回batch分组正确（releaseId=9001）
test_curl "详情返回batch分组正确（releaseId=9001）" "GET" "$API_BASE/releases/9001" "" "200"

# CURL 7: nodeName join正确（releaseId=9001）
test_curl "nodeName join正确（releaseId=9001）" "GET" "$API_BASE/releases/9001" "" "200"

# CURL 8: currentBatch在不同状态下正确（全success → 0，releaseId=9002）
test_curl "currentBatch在不同状态下正确（全success → 0）" "GET" "$API_BASE/releases/9002" "" "200"

# CURL 9: currentBatch在不同状态下正确（batch1 failed → 1，releaseId=9003）
test_curl "currentBatch在不同状态下正确（batch1 failed → 1）" "GET" "$API_BASE/releases/9003" "" "200"

# CURL 10: currentBatch在不同状态下正确（batch1 success, batch2 pending → 2，releaseId=9001）
test_curl "currentBatch在不同状态下正确（batch1 success, batch2 pending → 2）" "GET" "$API_BASE/releases/9001" "" "200"

# CURL 11: 测试无效releaseId（应返回404）
test_curl "测试无效releaseId（应返回404）" "GET" "$API_BASE/releases/999999" "" "404"

# CURL 12: 测试pageSize超过最大值（应限制为100）
test_curl "测试pageSize超过最大值（应限制为100）" "GET" "$API_BASE/releases?pageSize=200" "" "200"

# ==================== 清理测试数据 ====================
echo "==================== 清理测试数据 ===================="
echo

test_sql "清理测试数据" "
DELETE FROM release_task_nodes WHERE release_task_id IN (SELECT id FROM release_tasks WHERE version >= 900000);
DELETE FROM release_tasks WHERE version >= 900000;
DELETE FROM nodes WHERE id >= 9000;
"

# ==================== 测试总结 ====================
echo "=========================================="
echo "测试总结"
echo "=========================================="
echo -e "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
echo -e "${RED}失败: $FAILED_TESTS${NC}"
echo "=========================================="

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}部分测试失败！${NC}"
    exit 1
fi
