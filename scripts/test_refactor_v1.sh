#!/bin/bash
# T2-03改造清单v1.0验收测试脚本
# 测试version唯一性、Agent回滚能力、nginx -t错误详情、config_versions状态回写

set -e

BASE_URL="http://20.2.140.226:8080"
ADMIN_TOKEN="your_admin_token_here"

echo "========================================="
echo "T2-03改造清单v1.0 - 验收测试"
echo "========================================="
echo ""

# ========================================
# P0-01: version唯一性测试（数据库递增）
# ========================================
echo "【P0-01】测试version唯一性（数据库递增）"
echo "----------------------------------------"

echo "1. 并发50次apply，验证version递增无重复"
for i in {1..50}; do
  curl -s -X POST "$BASE_URL/api/v1/config/apply" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"nodeId": 1, "reason": "并发测试'$i'"}' &
done
wait
echo "✓ 50次并发apply完成"

echo ""
echo "SQL验证1: 检查version是否递增且无重复"
echo "SELECT version, COUNT(*) as cnt FROM config_versions GROUP BY version HAVING cnt > 1;"
echo "预期结果: Empty set (无重复version)"
echo ""

echo "SQL验证2: 检查version是否等于id"
echo "SELECT id, version FROM config_versions WHERE id != version LIMIT 10;"
echo "预期结果: Empty set (version应等于id)"
echo ""

# ========================================
# P0-02: Agent回滚能力测试
# ========================================
echo "【P0-02】测试Agent回滚能力（保留历史版本）"
echo "----------------------------------------"

echo "2. 第一次apply配置"
RESP1=$(curl -s -X POST "$BASE_URL/api/v1/config/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1, "reason": "第一次apply"}')
VERSION1=$(echo $RESP1 | jq -r '.data.version')
echo "✓ 第一次apply完成，version=$VERSION1"

sleep 3

echo "3. 第二次apply配置"
RESP2=$(curl -s -X POST "$BASE_URL/api/v1/config/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1, "reason": "第二次apply"}')
VERSION2=$(echo $RESP2 | jq -r '.data.version')
echo "✓ 第二次apply完成，version=$VERSION2"

echo ""
echo "Agent端验证（需在Agent服务器执行）:"
echo "ls -la /etc/nginx/cmdb/versions/"
echo "预期结果: 存在两个目录 $VERSION1/ 和 $VERSION2/"
echo ""
echo "ls -la /etc/nginx/cmdb/live"
echo "预期结果: live -> versions/$VERSION2/"
echo ""

echo "SQL验证3: 检查两个版本是否都存在"
echo "SELECT version, status FROM config_versions WHERE version IN ($VERSION1, $VERSION2);"
echo "预期结果: 两条记录"
echo ""

# ========================================
# P0-03: nginx -t错误详情测试
# ========================================
echo "【P0-03】测试nginx -t错误详情规范"
echo "----------------------------------------"

echo "4. 创建无效配置触发nginx -t失败"
echo "（需要手动修改数据库制造无效配置，或通过API创建无效域名）"
echo ""
echo "Agent端验证（需在Agent服务器执行）:"
echo "cat /etc/nginx/cmdb/meta/last_error.json"
echo "预期结果: 包含cmd、exitCode、stderr、error、time字段"
echo "示例:"
echo '{'
echo '  "version": 123,'
echo '  "cmd": "nginx -t -c /etc/nginx/nginx.conf",'
echo '  "exitCode": 1,'
echo '  "stderr": "nginx: [emerg] invalid server_name...",'
echo '  "error": "nginx test failed (exit code 1): nginx: [emerg]...",'
echo '  "time": "2026-01-23T12:00:00Z"'
echo '}'
echo ""

echo "SQL验证4: 检查失败版本的last_error字段"
echo "SELECT version, status, last_error FROM config_versions WHERE status = 'failed' LIMIT 5;"
echo "预期结果: last_error字段包含错误详情"
echo ""

# ========================================
# P1-01: redirect模式策略测试
# ========================================
echo "【P1-01】测试redirect模式策略（不生成upstream）"
echo "----------------------------------------"

echo "5. 创建redirect模式网站"
REDIRECT_WEBSITE=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "redirect_test",
    "status": "active",
    "originMode": "redirect",
    "redirectUrl": "https://example.com",
    "redirectStatusCode": 301
  }')
REDIRECT_WEBSITE_ID=$(echo $REDIRECT_WEBSITE | jq -r '.data.id')
echo "✓ 创建redirect网站，id=$REDIRECT_WEBSITE_ID"

echo "6. apply配置"
RESP_REDIRECT=$(curl -s -X POST "$BASE_URL/api/v1/config/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1, "reason": "测试redirect模式"}')
VERSION_REDIRECT=$(echo $RESP_REDIRECT | jq -r '.data.version')
echo "✓ apply完成，version=$VERSION_REDIRECT"

echo ""
echo "Agent端验证（需在Agent服务器执行）:"
echo "ls /etc/nginx/cmdb/live/upstreams/ | grep upstream_site_$REDIRECT_WEBSITE_ID"
echo "预期结果: 无输出（不应生成upstream文件）"
echo ""
echo "cat /etc/nginx/cmdb/live/servers/server_site_$REDIRECT_WEBSITE_ID.conf"
echo "预期结果: 包含 'return 301 https://example.com;' 语句"
echo ""

# ========================================
# P1-02: config_versions状态回写测试
# ========================================
echo "【P1-02】测试config_versions状态回写"
echo "----------------------------------------"

echo "7. 查询config_versions状态"
curl -s -X GET "$BASE_URL/api/v1/config/versions?nodeId=1&page=1&pageSize=10" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data.list[] | {version, status, lastError}'
echo ""

echo "SQL验证5: 统计各状态的版本数量"
echo "SELECT status, COUNT(*) as cnt FROM config_versions GROUP BY status;"
echo "预期结果: pending/applied/failed三种状态"
echo ""

echo "SQL验证6: 查询失败版本的详细信息"
echo "SELECT version, status, last_error, created_at FROM config_versions WHERE status = 'failed' ORDER BY version DESC LIMIT 5;"
echo "预期结果: last_error字段有内容"
echo ""

# ========================================
# P1-03: acme未签发策略测试
# ========================================
echo "【P1-03】测试acme未签发策略（自动降级为HTTP）"
echo "----------------------------------------"

echo "8. 创建ACME模式网站（未签发证书）"
ACME_WEBSITE=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme_test",
    "status": "active",
    "httpsEnabled": true,
    "certMode": "acme",
    "certificateId": 0
  }')
ACME_WEBSITE_ID=$(echo $ACME_WEBSITE | jq -r '.data.id')
echo "✓ 创建ACME网站（未签发证书），id=$ACME_WEBSITE_ID"

echo "9. apply配置"
RESP_ACME=$(curl -s -X POST "$BASE_URL/api/v1/config/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1, "reason": "测试ACME未签发"}')
VERSION_ACME=$(echo $RESP_ACME | jq -r '.data.version')
echo "✓ apply完成，version=$VERSION_ACME"

echo ""
echo "Agent端验证（需在Agent服务器执行）:"
echo "cat /etc/nginx/cmdb/live/servers/server_site_$ACME_WEBSITE_ID.conf | grep 'listen 443'"
echo "预期结果: 无输出（应降级为HTTP，不监听443）"
echo ""
echo "cat /etc/nginx/cmdb/live/servers/server_site_$ACME_WEBSITE_ID.conf | grep 'listen 80'"
echo "预期结果: 有输出（应监听80端口）"
echo ""

# ========================================
# 综合验收SQL
# ========================================
echo "========================================="
echo "综合验收SQL查询"
echo "========================================="
echo ""

echo "SQL验证7: 查询最近10个版本的完整信息"
echo "SELECT id, version, node_id, reason, status, last_error, applied_at, created_at FROM config_versions ORDER BY version DESC LIMIT 10;"
echo ""

echo "SQL验证8: 统计任务状态分布"
echo "SELECT status, COUNT(*) as cnt FROM agent_tasks WHERE type = 'apply_config' GROUP BY status;"
echo ""

echo "SQL验证9: 关联查询config_versions和agent_tasks"
echo "SELECT cv.version, cv.status as cv_status, at.status as task_status, at.last_error FROM config_versions cv LEFT JOIN agent_tasks at ON JSON_EXTRACT(at.payload, '$.version') = cv.version WHERE cv.node_id = 1 ORDER BY cv.version DESC LIMIT 10;"
echo ""

echo "SQL验证10: 查询version递增情况"
echo "SELECT MIN(version) as min_version, MAX(version) as max_version, COUNT(*) as total_count, MAX(version) - MIN(version) + 1 as expected_count FROM config_versions;"
echo "预期结果: total_count 接近 expected_count（允许有少量gap）"
echo ""

echo "========================================="
echo "验收测试完成"
echo "========================================="
echo ""
echo "请在Agent服务器执行以下命令验证目录结构:"
echo "  tree -L 3 /etc/nginx/cmdb/"
echo ""
echo "请在数据库执行上述SQL验证查询"
echo ""
