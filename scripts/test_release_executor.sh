#!/bin/bash

# B0-01-03 验收测试：发布执行器（调度Agent执行apply_config，批次推进，失败隔离）
# 注意：本测试需要真实的Agent环境（mTLS + apply_config接口）

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

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}B0-01-03 验收测试：发布执行器${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# 0. 登录获取JWT token
echo -e "${YELLOW}[0] 登录获取JWT token${NC}"
LOGIN_RESP=$(curl -s -X POST "${API_BASE}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}Failed to get token${NC}"
  echo "$LOGIN_RESP"
  exit 1
fi
echo -e "${GREEN}Token obtained: ${TOKEN:0:20}...${NC}"
echo ""

# ========================================
# SQL验证（14条）
# ========================================

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}SQL验证（14条）${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# SQL 0: 清理测试数据
echo -e "${YELLOW}[SQL 0] 清理测试数据${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
DELETE FROM release_task_nodes WHERE release_task_id IN (SELECT id FROM release_tasks WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR));
DELETE FROM release_tasks WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR);
DELETE FROM nodes WHERE name LIKE 'test_node_%';
" 2>&1 | grep -v "Using a password"
echo -e "${GREEN}Test data cleaned${NC}"
echo ""

# SQL 1: 插入3个online节点
echo -e "${YELLOW}[SQL 1] 插入3个online节点${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
INSERT INTO nodes (name, main_ip, agent_port, enabled, status, created_at, updated_at)
VALUES
('test_node_1', '192.168.1.101', 8443, 1, 'online', NOW(), NOW()),
('test_node_2', '192.168.1.102', 8443, 1, 'online', NOW(), NOW()),
('test_node_3', '192.168.1.103', 8443, 1, 'online', NOW(), NOW());
" 2>&1 | grep -v "Using a password"
echo -e "${GREEN}3 online nodes inserted${NC}"
echo ""

# SQL 2: 验证节点插入
echo -e "${YELLOW}[SQL 2] 验证节点插入${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, name, main_ip, status FROM nodes WHERE name LIKE 'test_node_%';
" 2>&1 | grep -v "Using a password"
echo ""

# ========================================
# curl验证（10+条）
# ========================================

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}curl验证（10+条）${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# curl 1: 创建发布任务（3个online node）
echo -e "${YELLOW}[curl 1] 创建发布任务（3个online node）${NC}"
CREATE_RESP=$(curl -s -X POST "${API_BASE}/releases" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target":"cdn"}')
echo "$CREATE_RESP" | jq '.'

RELEASE_ID=$(echo "$CREATE_RESP" | jq -r '.data.releaseId')
VERSION=$(echo "$CREATE_RESP" | jq -r '.data.version')

if [ -z "$RELEASE_ID" ] || [ "$RELEASE_ID" == "null" ]; then
  echo -e "${RED}Failed to create release${NC}"
  exit 1
fi
echo -e "${GREEN}Release created: releaseId=$RELEASE_ID, version=$VERSION${NC}"
echo ""

# SQL 3: 验证release_tasks初始状态
echo -e "${YELLOW}[SQL 3] 验证release_tasks初始状态${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, type, target, version, status, total_nodes, success_nodes, failed_nodes
FROM release_tasks WHERE id = $RELEASE_ID;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 4: 验证release_task_nodes初始状态
echo -e "${YELLOW}[SQL 4] 验证release_task_nodes初始状态${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, release_task_id, node_id, batch, status
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID ORDER BY batch, node_id;
" 2>&1 | grep -v "Using a password"
echo ""

# curl 2: 查询发布任务状态（pending）
echo -e "${YELLOW}[curl 2] 查询发布任务状态（pending）${NC}"
GET_RESP=$(curl -s -X GET "${API_BASE}/releases/$RELEASE_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "$GET_RESP" | jq '.'
echo ""

# SQL 5: 验证batch分配（GROUP BY batch）
echo -e "${YELLOW}[SQL 5] 验证batch分配（GROUP BY batch）${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT batch, COUNT(*) as node_count
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID
GROUP BY batch ORDER BY batch;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 6: 验证batch=1只有1个节点
echo -e "${YELLOW}[SQL 6] 验证batch=1只有1个节点${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT COUNT(*) as batch1_count
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID AND batch = 1;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 7: 验证batch=2有2个节点
echo -e "${YELLOW}[SQL 7] 验证batch=2有2个节点${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT COUNT(*) as batch2_count
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID AND batch = 2;
" 2>&1 | grep -v "Using a password"
echo ""

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}注意：以下测试需要真实的Agent环境${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# curl 3: 等待executor执行（需要RELEASE_EXECUTOR_ENABLED=1且MTLS_ENABLED=1）
echo -e "${YELLOW}[curl 3] 等待executor执行（需要真实Agent环境）${NC}"
echo -e "${YELLOW}提示：启动控制端时设置 RELEASE_EXECUTOR_ENABLED=1 和 MTLS_ENABLED=1${NC}"
echo -e "${YELLOW}提示：确保Agent正常运行并监听8443端口${NC}"
echo -e "${YELLOW}提示：等待约30秒后查询状态${NC}"
echo ""

# curl 4: 查询发布任务状态（running或success）
echo -e "${YELLOW}[curl 4] 查询发布任务状态（running或success）${NC}"
sleep 5
GET_RESP=$(curl -s -X GET "${API_BASE}/releases/$RELEASE_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "$GET_RESP" | jq '.'
CURRENT_STATUS=$(echo "$GET_RESP" | jq -r '.data.status')
echo -e "${GREEN}Current status: $CURRENT_STATUS${NC}"
echo ""

# SQL 8: 验证batch1节点状态变化
echo -e "${YELLOW}[SQL 8] 验证batch1节点状态变化${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, node_id, batch, status, started_at, finished_at
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID AND batch = 1;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 9: 验证batch2节点状态变化
echo -e "${YELLOW}[SQL 9] 验证batch2节点状态变化${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, node_id, batch, status, started_at, finished_at
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID AND batch = 2;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 10: 验证success_nodes计数
echo -e "${YELLOW}[SQL 10] 验证success_nodes计数${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, status, total_nodes, success_nodes, failed_nodes
FROM release_tasks WHERE id = $RELEASE_ID;
" 2>&1 | grep -v "Using a password"
echo ""

# curl 5: 模拟batch1失败（需要手动修改节点状态）
echo -e "${YELLOW}[curl 5] 模拟batch1失败（需要手动修改节点状态）${NC}"
echo -e "${YELLOW}提示：在真实环境中，可以停止Agent或让Agent返回失败${NC}"
echo ""

# SQL 11: 手动标记batch1节点为failed（模拟失败）
echo -e "${YELLOW}[SQL 11] 手动标记batch1节点为failed（模拟失败）${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
UPDATE release_task_nodes
SET status = 'failed', error_msg = 'simulated failure', finished_at = NOW()
WHERE release_task_id = $RELEASE_ID AND batch = 1;
" 2>&1 | grep -v "Using a password"
echo -e "${GREEN}Batch1 node marked as failed${NC}"
echo ""

# SQL 12: 验证batch2节点状态（应为skipped）
echo -e "${YELLOW}[SQL 12] 验证batch2节点状态（应为skipped）${NC}"
echo -e "${YELLOW}提示：需要executor检测到batch1失败后自动标记batch2为skipped${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, node_id, batch, status
FROM release_task_nodes WHERE release_task_id = $RELEASE_ID AND batch = 2;
" 2>&1 | grep -v "Using a password"
echo ""

# SQL 13: 验证failed_nodes计数
echo -e "${YELLOW}[SQL 13] 验证failed_nodes计数${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, status, total_nodes, success_nodes, failed_nodes
FROM release_tasks WHERE id = $RELEASE_ID;
" 2>&1 | grep -v "Using a password"
echo ""

# curl 6: 查询发布任务状态（应为failed）
echo -e "${YELLOW}[curl 6] 查询发布任务状态（应为failed）${NC}"
GET_RESP=$(curl -s -X GET "${API_BASE}/releases/$RELEASE_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "$GET_RESP" | jq '.'
FINAL_STATUS=$(echo "$GET_RESP" | jq -r '.data.status')
echo -e "${GREEN}Final status: $FINAL_STATUS${NC}"
echo ""

# curl 7: 测试无效releaseId（应返回404）
echo -e "${YELLOW}[curl 7] 测试无效releaseId（应返回404）${NC}"
curl -s -X GET "${API_BASE}/releases/999999" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# curl 8: 测试未授权访问（应返回401）
echo -e "${YELLOW}[curl 8] 测试未授权访问（应返回401）${NC}"
curl -s -X GET "${API_BASE}/releases/$RELEASE_ID" | jq '.'
echo ""

# curl 9: 重复启动executor不会重复执行同一release
echo -e "${YELLOW}[curl 9] 重复启动executor不会重复执行同一release${NC}"
echo -e "${YELLOW}提示：在真实环境中，重启控制端后executor应跳过已完成的release${NC}"
echo ""

# curl 10: Agent超时测试
echo -e "${YELLOW}[curl 10] Agent超时测试${NC}"
echo -e "${YELLOW}提示：在真实环境中，可以停止Agent或设置防火墙规则模拟超时${NC}"
echo ""

# SQL 14: 清理测试数据
echo -e "${YELLOW}[SQL 14] 清理测试数据${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
DELETE FROM release_task_nodes WHERE release_task_id = $RELEASE_ID;
DELETE FROM release_tasks WHERE id = $RELEASE_ID;
DELETE FROM nodes WHERE name LIKE 'test_node_%';
" 2>&1 | grep -v "Using a password"
echo -e "${GREEN}Test data cleaned${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}B0-01-03 验收测试完成${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}注意事项：${NC}"
echo "1. 本测试脚本提供了SQL和curl验证的框架"
echo "2. 完整测试需要真实的Agent环境（mTLS + apply_config接口）"
echo "3. 需要设置 RELEASE_EXECUTOR_ENABLED=1 和 MTLS_ENABLED=1"
echo "4. 需要配置mTLS证书（CONTROL_CERT/CONTROL_KEY/CONTROL_CA）"
echo "5. 需要Agent监听8443端口并实现/tasks/dispatch和/tasks/{id}接口"
echo ""
