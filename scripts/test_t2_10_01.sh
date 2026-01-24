#!/bin/bash
# T2-10-01 SQL验证脚本
# 验证domains/domain_dns_providers/domain_dns_zone_meta表的基础功能

set -e

MYSQL_CMD="mysql -uroot -S /data/mysql/run/mysql.sock cdn_control"

echo "========================================="
echo "T2-10-01 SQL验证脚本"
echo "========================================="
echo ""

echo "清理测试数据..."
$MYSQL_CMD -e "DELETE FROM domain_dns_zone_meta WHERE domain_id >= 9000;"
$MYSQL_CMD -e "DELETE FROM domain_dns_providers WHERE domain_id >= 9000;"
$MYSQL_CMD -e "DELETE FROM domains WHERE id >= 9000;"
$MYSQL_CMD -e "DELETE FROM api_keys WHERE id >= 9000;"
echo "清理完成"
echo ""

echo "========================================="
echo "准备测试数据: 创建api_keys记录"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO api_keys(id, name, provider, api_token) VALUES (9001, 'test-cloudflare-key', 'cloudflare', 'test_token_xxx');"
echo "插入成功: api_keys(id=9001, provider='cloudflare')"
echo ""

echo "========================================="
echo "验证1: 插入domains记录"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domains(id, domain, purpose, status) VALUES (9001, 'test-t2-10-01.com', 'cdn', 'active');"
echo "插入成功: domains(id=9001, domain='test-t2-10-01.com')"
echo ""

echo "========================================="
echo "验证2: 验证domains unique约束"
echo "========================================="
if $MYSQL_CMD -e "INSERT INTO domains(id, domain) VALUES (9002, 'test-t2-10-01.com');" 2>&1 | grep -q "Duplicate entry"; then
    echo "UNIQUE约束生效: 重复domain被拒绝"
else
    echo "错误: UNIQUE约束未生效"
    exit 1
fi
echo ""

echo "========================================="
echo "验证3: 插入domain_dns_providers记录（cloudflare）"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id, status) VALUES (9001, 'cloudflare', 'zone_xxx_cloudflare', 9001, 'active');"
echo "插入成功: domain_dns_providers(domain_id=9001, provider='cloudflare')"
echo ""

echo "========================================="
echo "验证4: 验证domain_dns_providers unique约束"
echo "========================================="
if $MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id) VALUES (9001, 'aliyun', 'zone_yyy', 9001);" 2>&1 | grep -q "Duplicate entry"; then
    echo "UNIQUE约束生效: 同一domain_id不能绑定多个provider"
else
    echo "错误: UNIQUE约束未生效"
    exit 1
fi
echo ""

echo "========================================="
echo "验证5: 插入domain_dns_providers记录（huawei）"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domains(id, domain) VALUES (9003, 'test-huawei.com');"
$MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id) VALUES (9003, 'huawei', 'zone_huawei', 9001);"
echo "插入成功: domain_dns_providers(domain_id=9003, provider='huawei')"
echo ""

echo "========================================="
echo "验证6: 插入domain_dns_zone_meta记录"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domain_dns_zone_meta(domain_id, name_servers_json, last_sync_at) VALUES (9001, '[\"ns1.cloudflare.com\",\"ns2.cloudflare.com\"]', NOW());"
echo "插入成功: domain_dns_zone_meta(domain_id=9001, name_servers_json='[\"ns1.cloudflare.com\",\"ns2.cloudflare.com\"]')"
echo ""

echo "========================================="
echo "验证7: 验证domain_dns_zone_meta unique约束"
echo "========================================="
if $MYSQL_CMD -e "INSERT INTO domain_dns_zone_meta(domain_id, name_servers_json, last_sync_at) VALUES (9001, '[\"ns3.cloudflare.com\"]', NOW());" 2>&1 | grep -q "Duplicate entry"; then
    echo "UNIQUE约束生效: 同一domain_id不能有多条zone_meta记录"
else
    echo "错误: UNIQUE约束未生效"
    exit 1
fi
echo ""

echo "========================================="
echo "验证8: 查询domain_dns_zone_meta记录"
echo "========================================="
$MYSQL_CMD -e "SELECT id, domain_id, name_servers_json, last_sync_at FROM domain_dns_zone_meta WHERE domain_id = 9001;"
echo ""

echo "========================================="
echo "验证9: 验证JSON字段格式"
echo "========================================="
RESULT=$($MYSQL_CMD -sN -e "SELECT JSON_VALID(name_servers_json) FROM domain_dns_zone_meta WHERE domain_id = 9001;")
if [ "$RESULT" = "1" ]; then
    echo "JSON格式验证通过"
else
    echo "错误: JSON格式无效"
    exit 1
fi
echo ""

echo "========================================="
echo "验证10: 测试purpose枚举值"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domains(id, domain, purpose) VALUES (9004, 'test-general.com', 'general');"
echo "插入成功: domains(id=9004, purpose='general')"
echo ""

echo "========================================="
echo "验证11: 测试status枚举值"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domains(id, domain, status) VALUES (9005, 'test-inactive.com', 'inactive');"
echo "插入成功: domains(id=9005, status='inactive')"
echo ""

echo "========================================="
echo "验证12: 测试所有provider类型"
echo "========================================="
$MYSQL_CMD -e "INSERT INTO domains(id, domain) VALUES (9006, 'test-aliyun.com');"
$MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id) VALUES (9006, 'aliyun', 'zone_aliyun', 9001);"
echo "插入成功: provider='aliyun'"

$MYSQL_CMD -e "INSERT INTO domains(id, domain) VALUES (9007, 'test-tencent.com');"
$MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id) VALUES (9007, 'tencent', 'zone_tencent', 9001);"
echo "插入成功: provider='tencent'"

$MYSQL_CMD -e "INSERT INTO domains(id, domain) VALUES (9008, 'test-manual.com');"
$MYSQL_CMD -e "INSERT INTO domain_dns_providers(domain_id, provider, provider_zone_id, api_key_id) VALUES (9008, 'manual', 'zone_manual', 9001);"
echo "插入成功: provider='manual'"
echo ""

echo "========================================="
echo "验证13: 查询所有测试数据"
echo "========================================="
echo "domains表:"
$MYSQL_CMD -e "SELECT id, domain, purpose, status FROM domains WHERE id >= 9000 ORDER BY id;"
echo ""
echo "domain_dns_providers表:"
$MYSQL_CMD -e "SELECT id, domain_id, provider, provider_zone_id FROM domain_dns_providers WHERE domain_id >= 9000 ORDER BY id;"
echo ""
echo "domain_dns_zone_meta表:"
$MYSQL_CMD -e "SELECT id, domain_id, name_servers_json, last_sync_at FROM domain_dns_zone_meta WHERE domain_id >= 9000 ORDER BY id;"
echo ""

echo "========================================="
echo "清理测试数据..."
echo "========================================="
$MYSQL_CMD -e "DELETE FROM domain_dns_zone_meta WHERE domain_id >= 9000;"
$MYSQL_CMD -e "DELETE FROM domain_dns_providers WHERE domain_id >= 9000;"
$MYSQL_CMD -e "DELETE FROM domains WHERE id >= 9000;"
$MYSQL_CMD -e "DELETE FROM api_keys WHERE id >= 9000;"
echo "清理完成"
echo ""

echo "========================================="
echo "所有验证通过！"
echo "========================================="
