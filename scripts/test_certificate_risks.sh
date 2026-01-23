#!/bin/bash

# T2-08: 证书与网站风险预检 + 告警体系 - 验收测试脚本
# 包含20+条CURL测试和15+条SQL验证

set -e

API_BASE="http://localhost:8080/api/v1"
TOKEN=""

echo "========================================="
echo "T2-08 Certificate Risks - Acceptance Test"
echo "========================================="
echo ""

# ============================================
# CURL测试（20+条）
# ============================================

echo "=== CURL Tests ==="
echo ""

# 1. 登录获取JWT token
echo "[CURL-01] Login to get JWT token"
LOGIN_RESP=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  echo "  ❌ Failed to get token"
  exit 1
fi
echo "  ✓ Token obtained: ${TOKEN:0:20}..."
echo ""

# 2. 查询全局风险列表（无过滤）
echo "[CURL-02] List all risks (no filter)"
curl -s -X GET "$API_BASE/risks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 3. 查询全局风险列表（按level过滤）
echo "[CURL-03] List risks filtered by level=critical"
curl -s -X GET "$API_BASE/risks?level=critical" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 4. 查询全局风险列表（按risk_type过滤）
echo "[CURL-04] List risks filtered by risk_type=domain_mismatch"
curl -s -X GET "$API_BASE/risks?risk_type=domain_mismatch" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 5. 查询全局风险列表（按status过滤）
echo "[CURL-05] List risks filtered by status=active"
curl -s -X GET "$API_BASE/risks?status=active" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 6. 查询全局风险列表（分页）
echo "[CURL-06] List risks with pagination (page=1, page_size=10)"
curl -s -X GET "$API_BASE/risks?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {total, page, page_size, total_pages}'
echo ""

# 7. 查询网站风险列表（有效网站）
echo "[CURL-07] List website risks (website_id=1)"
curl -s -X GET "$API_BASE/websites/1/risks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {website_id, count}'
echo ""

# 8. 查询网站风险列表（无效网站）
echo "[CURL-08] List website risks (invalid website_id=99999, should return empty)"
curl -s -X GET "$API_BASE/websites/99999/risks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {website_id, count}'
echo ""

# 9. 查询证书风险列表（有效证书）
echo "[CURL-09] List certificate risks (certificate_id=1)"
curl -s -X GET "$API_BASE/certificates/1/risks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {certificate_id, count}'
echo ""

# 10. 查询证书风险列表（无效证书）
echo "[CURL-10] List certificate risks (invalid certificate_id=99999, should return empty)"
curl -s -X GET "$API_BASE/certificates/99999/risks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {certificate_id, count}'
echo ""

# 11. 前置预检（select模式，partial覆盖，应返回ok=false）
echo "[CURL-11] Precheck HTTPS (select mode, partial coverage, should return ok=false)"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select","certificate_id":1}' | jq '.data | {ok, risks: (.risks | length)}'
echo ""

# 12. 检查预检响应包含risk类型
echo "[CURL-12] Check precheck response contains risk types"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select","certificate_id":1}' | jq '.data.risks[0] | {type, level}'
echo ""

# 13. 检查预检响应包含detail
echo "[CURL-13] Check precheck response contains detail"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select","certificate_id":1}' | jq '.data.risks[0].detail | keys'
echo ""

# 14. 前置预检（acme模式，绕过校验，应返回ok=true）
echo "[CURL-14] Precheck HTTPS (acme mode, bypass check, should return ok=true)"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"acme"}' | jq '.data | {ok, risks: (.risks | length)}'
echo ""

# 15. 解决风险（有效风险ID）
echo "[CURL-15] Resolve risk (valid risk_id=1)"
curl -s -X POST "$API_BASE/risks/1/resolve" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {risk_id, message}'
echo ""

# 16. 验证风险已解决（查询resolved状态）
echo "[CURL-16] Verify risk is resolved (status=resolved)"
curl -s -X GET "$API_BASE/risks?status=resolved" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 17. 重复解决已resolved的风险（应失败）
echo "[CURL-17] Resolve already resolved risk (should fail)"
curl -s -X POST "$API_BASE/risks/1/resolve" \
  -H "Authorization: Bearer $TOKEN" | jq '.code'
echo ""

# 18. 解决风险（无效风险ID，应返回404）
echo "[CURL-18] Resolve risk (invalid risk_id=99999, should return 404)"
curl -s -X POST "$API_BASE/risks/99999/resolve" \
  -H "Authorization: Bearer $TOKEN" | jq '.code'
echo ""

# 19. 测试未授权访问风险列表
echo "[CURL-19] Test unauthorized access to risks list (should return 401)"
curl -s -X GET "$API_BASE/risks" | jq '.code'
echo ""

# 20. 测试未授权访问预检API
echo "[CURL-20] Test unauthorized access to precheck API (should return 401)"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select","certificate_id":1}' | jq '.code'
echo ""

# 21. 测试缺失参数的预检请求
echo "[CURL-21] Test precheck with missing certificate_id (select mode, should fail)"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select"}' | jq '.code'
echo ""

# 22. 测试invalid cert_mode
echo "[CURL-22] Test precheck with invalid cert_mode (should fail)"
curl -s -X POST "$API_BASE/websites/1/precheck/https" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"invalid"}' | jq '.code'
echo ""

# 23. 查询风险列表（按certificate_id过滤）
echo "[CURL-23] List risks filtered by certificate_id=1"
curl -s -X GET "$API_BASE/risks?certificate_id=1" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 24. 查询风险列表（按website_id过滤）
echo "[CURL-24] List risks filtered by website_id=1"
curl -s -X GET "$API_BASE/risks?website_id=1" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.risks | length'
echo ""

# 25. 测试分页边界（page=0）
echo "[CURL-25] Test pagination boundary (page=0, should use default page=1)"
curl -s -X GET "$API_BASE/risks?page=0&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.page'
echo ""

echo ""
echo "=== CURL Tests Completed ==="
echo ""

# ============================================
# SQL验证（15+条）
# ============================================

echo "=== SQL Validation ==="
echo ""

# 注意：以下SQL命令需要在数据库服务器上执行
# 这里提供SQL语句供手动验证

cat << 'EOF'
-- SQL验证脚本（需要在数据库服务器上执行）

-- 1. 验证certificate_risks表存在
DESCRIBE certificate_risks;

-- 2. 验证risk_type字段枚举值
SHOW COLUMNS FROM certificate_risks LIKE 'risk_type';

-- 3. 验证level字段枚举值
SHOW COLUMNS FROM certificate_risks LIKE 'level';

-- 4. 验证status字段枚举值
SHOW COLUMNS FROM certificate_risks LIKE 'status';

-- 5. 验证detail字段类型为JSON
SHOW COLUMNS FROM certificate_risks LIKE 'detail';

-- 6. 验证唯一约束存在
SHOW INDEXES FROM certificate_risks WHERE Key_name = 'uk_risk';

-- 7. 统计active状态的风险数量
SELECT COUNT(*) AS active_risks FROM certificate_risks WHERE status = 'active';

-- 8. 统计resolved状态的风险数量
SELECT COUNT(*) AS resolved_risks FROM certificate_risks WHERE status = 'resolved';

-- 9. 统计各类型风险数量
SELECT risk_type, COUNT(*) AS count 
FROM certificate_risks 
WHERE status = 'active'
GROUP BY risk_type;

-- 10. 统计各级别风险数量
SELECT level, COUNT(*) AS count 
FROM certificate_risks 
WHERE status = 'active'
GROUP BY level;

-- 11. 检查domain_mismatch风险的detail内容
SELECT id, detail 
FROM certificate_risks 
WHERE risk_type = 'domain_mismatch' 
LIMIT 1;

-- 12. 检查cert_expiring风险的detail内容
SELECT id, detail 
FROM certificate_risks 
WHERE risk_type = 'cert_expiring' 
LIMIT 1;

-- 13. 检查weak_coverage风险的detail内容
SELECT id, detail 
FROM certificate_risks 
WHERE risk_type = 'weak_coverage' 
LIMIT 1;

-- 14. 验证风险幂等（同一风险不重复生成）
SELECT risk_type, certificate_id, website_id, COUNT(*) AS count
FROM certificate_risks
WHERE status = 'active'
GROUP BY risk_type, certificate_id, website_id
HAVING count > 1;

-- 15. 验证resolved_at字段在resolved状态时不为空
SELECT COUNT(*) AS invalid_resolved_risks
FROM certificate_risks
WHERE status = 'resolved' AND resolved_at IS NULL;

-- 16. 验证detected_at字段不为空
SELECT COUNT(*) AS invalid_detected_risks
FROM certificate_risks
WHERE detected_at IS NULL;

-- 17. 统计关联证书的风险数量
SELECT certificate_id, COUNT(*) AS risk_count
FROM certificate_risks
WHERE certificate_id IS NOT NULL AND status = 'active'
GROUP BY certificate_id
ORDER BY risk_count DESC
LIMIT 10;

-- 18. 统计关联网站的风险数量
SELECT website_id, COUNT(*) AS risk_count
FROM certificate_risks
WHERE website_id IS NOT NULL AND status = 'active'
GROUP BY website_id
ORDER BY risk_count DESC
LIMIT 10;

-- 19. 验证风险状态流转（resolved风险的detected_at < resolved_at）
SELECT COUNT(*) AS invalid_time_flow
FROM certificate_risks
WHERE status = 'resolved' 
  AND resolved_at IS NOT NULL 
  AND detected_at >= resolved_at;

-- 20. 检查最近检测到的风险
SELECT id, risk_type, level, detected_at
FROM certificate_risks
WHERE status = 'active'
ORDER BY detected_at DESC
LIMIT 10;

EOF

echo ""
echo "=== SQL Validation Statements Generated ==="
echo "Please execute the above SQL statements on the database server."
echo ""

echo "========================================="
echo "T2-08 Acceptance Test Completed"
echo "========================================="
