#!/bin/bash

# B0-01-01: 发布模型与表结构 - 验收测试脚本
# 包含6+条SQL验证和插入测试

set -e

echo "========================================="
echo "B0-01-01 Release Models - Acceptance Test"
echo "========================================="
echo ""

# 数据库连接信息（根据实际环境修改）
DB_HOST="${DB_HOST:-20.2.140.226}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-root123}"
DB_NAME="${DB_NAME:-cmdb}"

echo "Database: $DB_HOST/$DB_NAME"
echo ""

# ============================================
# SQL验证（10条）
# ============================================

echo "=== SQL Validation ==="
echo ""

# 1. 验证release_tasks表存在
echo "[SQL-01] Verify release_tasks table exists"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW TABLES LIKE 'release_tasks';" 2>/dev/null
echo ""

# 2. 验证release_task_nodes表存在
echo "[SQL-02] Verify release_task_nodes table exists"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW TABLES LIKE 'release_task_nodes';" 2>/dev/null
echo ""

# 3. 查看release_tasks表结构
echo "[SQL-03] Show release_tasks table structure"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW CREATE TABLE release_tasks\G" 2>/dev/null
echo ""

# 4. 查看release_task_nodes表结构
echo "[SQL-04] Show release_task_nodes table structure"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW CREATE TABLE release_task_nodes\G" 2>/dev/null
echo ""

# 5. 验证release_tasks表索引
echo "[SQL-05] Show release_tasks indexes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW INDEX FROM release_tasks;" 2>/dev/null
echo ""

# 6. 验证release_task_nodes表索引
echo "[SQL-06] Show release_task_nodes indexes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "SHOW INDEX FROM release_task_nodes;" 2>/dev/null
echo ""

# 7. 插入一条release_task（正常插入）
echo "[SQL-07] Insert a release_task (should succeed)"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO release_tasks (type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
VALUES ('apply_config', 'cdn', 1001, 'pending', 10, 0, 0, NOW(), NOW());
" 2>/dev/null
echo "  ✓ Insert succeeded"
echo ""

# 8. 插入两条release_task_nodes（正常插入）
echo "[SQL-08] Insert two release_task_nodes (should succeed)"
TASK_ID=$(mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -se "SELECT id FROM release_tasks WHERE version=1001 LIMIT 1;" 2>/dev/null)
echo "  Task ID: $TASK_ID"

mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
VALUES ($TASK_ID, 1, 1, 'pending', NOW(), NOW());
" 2>/dev/null
echo "  ✓ Insert node 1 succeeded"

mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
VALUES ($TASK_ID, 2, 1, 'pending', NOW(), NOW());
" 2>/dev/null
echo "  ✓ Insert node 2 succeeded"
echo ""

# 9. 测试version唯一约束（应失败）
echo "[SQL-09] Test version unique constraint (should fail)"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO release_tasks (type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
VALUES ('apply_config', 'cdn', 1001, 'pending', 10, 0, 0, NOW(), NOW());
" 2>&1 | grep -q "Duplicate entry" && echo "  ✓ Unique constraint works (duplicate rejected)" || echo "  ✗ Unique constraint failed"
echo ""

# 10. 测试release_task_id+node_id唯一约束（应失败）
echo "[SQL-10] Test (release_task_id, node_id) unique constraint (should fail)"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
VALUES ($TASK_ID, 1, 2, 'pending', NOW(), NOW());
" 2>&1 | grep -q "Duplicate entry" && echo "  ✓ Unique constraint works (duplicate rejected)" || echo "  ✗ Unique constraint failed"
echo ""

# 11. 验证插入的数据
echo "[SQL-11] Verify inserted data"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT * FROM release_tasks WHERE version=1001;
" 2>/dev/null
echo ""

echo "[SQL-12] Verify inserted nodes"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT * FROM release_task_nodes WHERE release_task_id=$TASK_ID;
" 2>/dev/null
echo ""

# 12. 清理测试数据
echo "[SQL-13] Cleanup test data"
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS $DB_NAME -e "
DELETE FROM release_task_nodes WHERE release_task_id=$TASK_ID;
DELETE FROM release_tasks WHERE version=1001;
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
echo "B0-01-01 Acceptance Test Completed"
echo "========================================="
