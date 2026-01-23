#!/bin/bash
# T2-04 DNS Worker验收测试脚本
# 测试DNS记录从pending到active/error的完整生命周期

set -e

BASE_URL="http://20.2.140.226:8080"
ADMIN_TOKEN="your_admin_token_here"  # 需要替换为真实token

echo "========================================="
echo "T2-04 DNS Worker 验收测试"
echo "========================================="
echo ""

# 前置条件：需要先创建domain和domain_dns_provider
echo "【前置条件】请确保已创建："
echo "1. domain (id=1, domain='example.com', status='active')"
echo "2. domain_dns_provider (domain_id=1, provider='cloudflare', status='active', api_key_id=1)"
echo "3. api_keys (id=1, api_token='cloudflare_api_token')"
echo ""
read -p "按Enter继续测试..."
echo ""

# ==========================================
# 测试1: 创建DNS记录（A记录）
# ==========================================
echo "【测试1】创建DNS记录（A记录，node_group owner）"
curl -X POST "${BASE_URL}/api/v1/dns/records/create" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "domainId": 1,
    "type": "A",
    "name": "www",
    "value": "1.2.3.4",
    "ttl": 120,
    "ownerType": "node_group",
    "ownerId": 1
  }' | jq .
echo ""

# ==========================================
# 测试2: 创建DNS记录（CNAME记录）
# ==========================================
echo "【测试2】创建DNS记录（CNAME记录，website_domain owner）"
curl -X POST "${BASE_URL}/api/v1/dns/records/create" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "domainId": 1,
    "type": "CNAME",
    "name": "cdn",
    "value": "cdn.example.net",
    "ttl": 300,
    "ownerType": "website_domain",
    "ownerId": 1
  }' | jq .
echo ""

# ==========================================
# 测试3: 创建DNS记录（TXT记录，ACME challenge）
# ==========================================
echo "【测试3】创建DNS记录（TXT记录，acme_challenge owner）"
curl -X POST "${BASE_URL}/api/v1/dns/records/create" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "domainId": 1,
    "type": "TXT",
    "name": "_acme-challenge",
    "value": "test-acme-token-12345",
    "ttl": 60,
    "ownerType": "acme_challenge",
    "ownerId": 1
  }' | jq .
echo ""

# ==========================================
# 测试4: 创建DNS记录（@根域名）
# ==========================================
echo "【测试4】创建DNS记录（@根域名，line_group owner）"
curl -X POST "${BASE_URL}/api/v1/dns/records/create" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "domainId": 1,
    "type": "A",
    "name": "@",
    "value": "5.6.7.8",
    "ttl": 120,
    "ownerType": "line_group",
    "ownerId": 1
  }' | jq .
echo ""

# ==========================================
# 测试5: 创建DNS记录（a.b子域名）
# ==========================================
echo "【测试5】创建DNS记录（a.b子域名）"
curl -X POST "${BASE_URL}/api/v1/dns/records/create" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "domainId": 1,
    "type": "A",
    "name": "a.b",
    "value": "9.10.11.12",
    "ttl": 120,
    "ownerType": "node_group",
    "ownerId": 2
  }' | jq .
echo ""

# ==========================================
# 测试6: 查询DNS记录列表（全部）
# ==========================================
echo "【测试6】查询DNS记录列表（全部）"
curl -X GET "${BASE_URL}/api/v1/dns/records?page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试7: 查询DNS记录列表（按domainId筛选）
# ==========================================
echo "【测试7】查询DNS记录列表（按domainId=1筛选）"
curl -X GET "${BASE_URL}/api/v1/dns/records?domainId=1&page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试8: 查询DNS记录列表（按status筛选）
# ==========================================
echo "【测试8】查询DNS记录列表（按status=pending筛选）"
curl -X GET "${BASE_URL}/api/v1/dns/records?status=pending&page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试9: 查询DNS记录列表（按ownerType筛选）
# ==========================================
echo "【测试9】查询DNS记录列表（按ownerType=node_group筛选）"
curl -X GET "${BASE_URL}/api/v1/dns/records?ownerType=node_group&page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试10: 查询单条DNS记录
# ==========================================
echo "【测试10】查询单条DNS记录（id=1）"
curl -X GET "${BASE_URL}/api/v1/dns/records/1" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试11: 等待Worker处理（40秒）
# ==========================================
echo "【测试11】等待DNS Worker处理记录（40秒）..."
echo "Worker将在40秒内处理pending记录并同步到Cloudflare"
sleep 40
echo "等待完成"
echo ""

# ==========================================
# 测试12: 查询DNS记录状态（应该变为active）
# ==========================================
echo "【测试12】查询DNS记录状态（应该变为active）"
curl -X GET "${BASE_URL}/api/v1/dns/records?status=active&page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试13: 查询DNS记录状态（检查error记录）
# ==========================================
echo "【测试13】查询DNS记录状态（检查error记录）"
curl -X GET "${BASE_URL}/api/v1/dns/records?status=error&page=1&pageSize=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
echo ""

# ==========================================
# 测试14: 手动重试失败记录
# ==========================================
echo "【测试14】手动重试失败记录（假设id=1失败）"
curl -X POST "${BASE_URL}/api/v1/dns/records/retry" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1
  }' | jq .
echo ""

# ==========================================
# 测试15: 标记DNS记录为删除（desired_state=absent）
# ==========================================
echo "【测试15】标记DNS记录为删除（id=1,2）"
curl -X POST "${BASE_URL}/api/v1/dns/records/delete" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [1, 2]
  }' | jq .
echo ""

# ==========================================
# 测试16: 查询待删除记录（desired_state=absent）
# ==========================================
echo "【测试16】查询待删除记录（desired_state=absent）"
echo "SQL: SELECT * FROM domain_dns_records WHERE desired_state='absent';"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, desired_state, provider_record_id FROM domain_dns_records WHERE desired_state='absent';"
echo ""

# ==========================================
# 测试17: 等待Worker删除记录（40秒）
# ==========================================
echo "【测试17】等待DNS Worker删除记录（40秒）..."
sleep 40
echo "等待完成"
echo ""

# ==========================================
# 测试18: 验证记录已从数据库删除
# ==========================================
echo "【测试18】验证记录已从数据库删除"
echo "SQL: SELECT * FROM domain_dns_records WHERE id IN (1, 2);"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT * FROM domain_dns_records WHERE id IN (1, 2);"
echo "（应该返回空结果）"
echo ""

# ==========================================
# SQL验证（12条）
# ==========================================
echo "========================================="
echo "SQL验证（12条）"
echo "========================================="
echo ""

echo "【SQL 1】查询所有pending状态的DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status FROM domain_dns_records WHERE status='pending';"
echo ""

echo "【SQL 2】查询所有active状态的DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, provider_record_id FROM domain_dns_records WHERE status='active';"
echo ""

echo "【SQL 3】查询所有error状态的DNS记录（含错误信息）"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, last_error, retry_count, next_retry_at FROM domain_dns_records WHERE status='error';"
echo ""

echo "【SQL 4】查询所有desired_state=absent的DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, desired_state, provider_record_id FROM domain_dns_records WHERE desired_state='absent';"
echo ""

echo "【SQL 5】按ownerType统计DNS记录数量"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT owner_type, COUNT(*) as count FROM domain_dns_records GROUP BY owner_type;"
echo ""

echo "【SQL 6】按status统计DNS记录数量"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT status, COUNT(*) as count FROM domain_dns_records GROUP BY status;"
echo ""

echo "【SQL 7】查询retry_count >= 5的DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, retry_count, last_error FROM domain_dns_records WHERE retry_count >= 5;"
echo ""

echo "【SQL 8】查询next_retry_at不为空的DNS记录（等待重试）"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, retry_count, next_retry_at FROM domain_dns_records WHERE next_retry_at IS NOT NULL ORDER BY next_retry_at;"
echo ""

echo "【SQL 9】查询provider_record_id不为空的DNS记录（已同步到Cloudflare）"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, provider_record_id FROM domain_dns_records WHERE provider_record_id IS NOT NULL AND provider_record_id != '';"
echo ""

echo "【SQL 10】关联查询domain和DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT d.domain, r.type, r.name, r.value, r.status FROM domain_dns_records r JOIN domains d ON r.domain_id = d.id WHERE d.status='active';"
echo ""

echo "【SQL 11】查询最近创建的10条DNS记录"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, created_at FROM domain_dns_records ORDER BY created_at DESC LIMIT 10;"
echo ""

echo "【SQL 12】查询retry_count >= 10的DNS记录（停止自动重试）"
mysql -h 20.2.140.226 -u root -p123456 cmdb -e "SELECT id, domain_id, type, name, value, status, retry_count, next_retry_at, last_error FROM domain_dns_records WHERE retry_count >= 10;"
echo ""

echo "========================================="
echo "T2-04 DNS Worker 验收测试完成"
echo "========================================="
echo ""

echo "【验收要点】"
echo "1. 创建DNS记录后status=pending"
echo "2. Worker处理后status=active（成功）或error（失败）"
echo "3. active记录的provider_record_id不为空"
echo "4. error记录的last_error包含错误信息"
echo "5. error记录的retry_count递增"
echo "6. error记录的next_retry_at按退避策略计算"
echo "7. retry_count >= 10的记录next_retry_at为null（停止自动重试）"
echo "8. desired_state=absent的记录被Worker删除"
echo "9. 删除后记录从数据库硬删除"
echo "10. name规则正确：@ → example.com, www → www.example.com, a.b → a.b.example.com"
echo "11. 支持4种ownerType：node_group, line_group, website_domain, acme_challenge"
echo "12. 支持4种recordType：A, AAAA, CNAME, TXT"
echo ""
