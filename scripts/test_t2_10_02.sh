#!/bin/bash
# T2-10-02 验收测试脚本
# 验证Domain Sync功能

set -e

MYSQL_CMD="mysql -uroot -S /data/mysql/run/mysql.sock cdn_control"
API_BASE="http://localhost:8080/api/v1"

echo "========================================="
echo "T2-10-02 验收测试脚本"
echo "========================================="
echo ""

echo "注意: 本测试需要真实的Cloudflare API Token"
echo "如果没有API Token，将只进行SQL结构验证"
echo ""

# 检查是否提供了API Token
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo "警告: 未设置CLOUDFLARE_API_TOKEN环境变量"
    echo "将跳过实际API调用测试，只进行SQL结构验证"
    echo ""
    SKIP_API_TEST=1
else
    echo "检测到CLOUDFLARE_API_TOKEN，将进行完整测试"
    echo ""
    SKIP_API_TEST=0
fi

echo "========================================="
echo "验证1: 检查domains表purpose枚举"
echo "========================================="
RESULT=$($MYSQL_CMD -sN -e "SHOW COLUMNS FROM domains LIKE 'purpose';")
if echo "$RESULT" | grep -q "enum('unset','cdn','general')"; then
    echo "通过: domains.purpose枚举包含unset"
else
    echo "失败: domains.purpose枚举不正确"
    exit 1
fi
echo ""

echo "========================================="
echo "验证2: 检查domains表purpose默认值"
echo "========================================="
DEFAULT=$($MYSQL_CMD -sN -e "SHOW COLUMNS FROM domains LIKE 'purpose';" | awk '{print $4}')
if [ "$DEFAULT" = "unset" ]; then
    echo "通过: domains.purpose默认值为unset"
else
    echo "失败: domains.purpose默认值不是unset，实际为: $DEFAULT"
    exit 1
fi
echo ""

echo "========================================="
echo "验证3: 准备测试数据（api_keys）"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    # 清理旧的测试数据
    $MYSQL_CMD -e "DELETE FROM api_keys WHERE id >= 9000;"
    
    # 插入测试用的Cloudflare API Key
    $MYSQL_CMD -e "INSERT INTO api_keys(id, name, provider, api_token) VALUES (9001, 'test-cloudflare-key', 'cloudflare', '$CLOUDFLARE_API_TOKEN');"
    echo "插入成功: api_keys(id=9001, provider='cloudflare')"
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "验证4: 调用同步API"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    # 获取JWT Token
    LOGIN_RESP=$(curl -s -X POST $API_BASE/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"admin123"}')
    
    JWT_TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    
    if [ -z "$JWT_TOKEN" ]; then
        echo "失败: 无法获取JWT Token"
        exit 1
    fi
    
    echo "JWT Token已获取"
    
    # 调用同步API
    SYNC_RESP=$(curl -s -X POST $API_BASE/domains/sync \
      -H "Authorization: Bearer $JWT_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"apiKeyId":9001}')
    
    echo "同步响应: $SYNC_RESP"
    
    # 检查响应code
    CODE=$(echo $SYNC_RESP | grep -o '"code":[0-9]*' | cut -d':' -f2)
    if [ "$CODE" != "0" ]; then
        echo "失败: 同步API返回错误 code=$CODE"
        exit 1
    fi
    
    echo "通过: 同步API调用成功"
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "验证5: SQL验证 - 新域名purpose=unset"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    # 查询最近同步的域名
    RECENT_DOMAINS=$($MYSQL_CMD -sN -e "SELECT domain, purpose FROM domains ORDER BY created_at DESC LIMIT 5;")
    echo "最近同步的域名:"
    echo "$RECENT_DOMAINS"
    
    # 检查是否有purpose=unset的域名
    if echo "$RECENT_DOMAINS" | grep -q "unset"; then
        echo "通过: 发现purpose=unset的域名"
    else
        echo "警告: 未发现purpose=unset的域名（可能所有域名都已存在）"
    fi
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "验证6: SQL验证 - NS已写入"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    # 查询domain_dns_zone_meta
    NS_COUNT=$($MYSQL_CMD -sN -e "SELECT COUNT(*) FROM domain_dns_zone_meta;")
    echo "domain_dns_zone_meta记录数: $NS_COUNT"
    
    if [ "$NS_COUNT" -gt 0 ]; then
        echo "通过: NS已写入domain_dns_zone_meta"
        
        # 显示一条示例
        SAMPLE=$($MYSQL_CMD -e "SELECT domain_id, name_servers_json, last_sync_at FROM domain_dns_zone_meta LIMIT 1;")
        echo "示例记录:"
        echo "$SAMPLE"
    else
        echo "失败: domain_dns_zone_meta为空"
        exit 1
    fi
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "验证7: SQL验证 - domain_dns_providers正确"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    # 查询domain_dns_providers
    PROVIDER_COUNT=$($MYSQL_CMD -sN -e "SELECT COUNT(*) FROM domain_dns_providers WHERE provider='cloudflare';")
    echo "Cloudflare provider记录数: $PROVIDER_COUNT"
    
    if [ "$PROVIDER_COUNT" -gt 0 ]; then
        echo "通过: domain_dns_providers已创建"
        
        # 显示一条示例
        SAMPLE=$($MYSQL_CMD -e "SELECT domain_id, provider, provider_zone_id, api_key_id FROM domain_dns_providers WHERE provider='cloudflare' LIMIT 1;")
        echo "示例记录:"
        echo "$SAMPLE"
    else
        echo "失败: domain_dns_providers为空"
        exit 1
    fi
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "验证8: 清理测试数据"
echo "========================================="
if [ $SKIP_API_TEST -eq 0 ]; then
    $MYSQL_CMD -e "DELETE FROM api_keys WHERE id >= 9000;"
    echo "清理完成"
else
    echo "跳过: 无API Token"
fi
echo ""

echo "========================================="
echo "测试完成"
echo "========================================="
if [ $SKIP_API_TEST -eq 1 ]; then
    echo "注意: 由于缺少API Token，部分测试被跳过"
    echo "要进行完整测试，请设置CLOUDFLARE_API_TOKEN环境变量"
    echo ""
    echo "示例:"
    echo "export CLOUDFLARE_API_TOKEN='your_token_here'"
    echo "bash scripts/test_t2_10_02.sh"
fi
