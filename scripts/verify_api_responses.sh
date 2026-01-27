#!/bin/bash
# 验证所有接口的响应字段是否符合 lowerCamelCase 规范

set -e

API_BASE="http://20.2.140.226:8080"
REPORT_FILE="/tmp/api_response_verification.txt"

echo "========================================="
echo "API 响应字段验证"
echo "========================================="
echo ""

# 清空报告文件
> "$REPORT_FILE"

# 登录获取 Token
echo "正在登录..."
TOKEN=$(curl -s -X POST "$API_BASE/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' | \
    grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "✗ 登录失败"
    exit 1
fi

echo "✓ 登录成功"
echo ""

# 检查函数：验证 JSON 中是否包含 snake_case 字段
check_response() {
    local endpoint="$1"
    local method="$2"
    local data="$3"
    local name="$4"
    
    echo "检查: $name"
    echo "  端点: $method $endpoint" >> "$REPORT_FILE"
    
    if [ "$method" = "GET" ]; then
        RESPONSE=$(curl -s -X GET "$API_BASE$endpoint" -H "Authorization: Bearer $TOKEN")
    else
        RESPONSE=$(curl -s -X POST "$API_BASE$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    # 检查是否包含 snake_case 字段
    SNAKE_FIELDS=$(echo "$RESPONSE" | grep -oE '"[a-z]+_[a-z_]+"' || true)
    
    if [ -n "$SNAKE_FIELDS" ]; then
        echo "  ✗ 发现 snake_case 字段:" >> "$REPORT_FILE"
        echo "$SNAKE_FIELDS" | sort -u | while read -r field; do
            echo "    $field" >> "$REPORT_FILE"
        done
        echo "" >> "$REPORT_FILE"
        return 1
    else
        echo "  ✓ 通过" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        return 0
    fi
}

TOTAL_CHECKS=0
PASSED_CHECKS=0

echo "开始验证接口..."
echo ""

# P0-1: Nodes
echo "=== P0-1: Nodes ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++))
check_response "/api/v1/nodes?page=1&pageSize=10" "GET" "" "Nodes List" && ((PASSED_CHECKS++)) || true

((TOTAL_CHECKS++))
check_response "/api/v1/nodes/7" "GET" "" "Node Get" && ((PASSED_CHECKS++)) || true

# P0-2: Domains
echo "=== P0-2: Domains ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++))
check_response "/api/v1/domains?page=1&pageSize=10" "GET" "" "Domains List" && ((PASSED_CHECKS++)) || true

# P0-2: DNS Records
echo "=== P0-2: DNS Records ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++))
check_response "/api/v1/dns/records?page=1&pageSize=10" "GET" "" "DNS Records List" && ((PASSED_CHECKS++)) || true

# P0-3: Certificates
echo "=== P0-3: Certificates ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++))
check_response "/api/v1/cert/lifecycle?page=1&pageSize=10" "GET" "" "Certificates Lifecycle" && ((PASSED_CHECKS++)) || true

# P0-4: API Keys
echo "=== P0-4: API Keys ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++))
check_response "/api/v1/api-keys?page=1&pageSize=10" "GET" "" "API Keys List" && ((PASSED_CHECKS++)) || true

echo ""
echo "========================================="
echo "验证完成"
echo "========================================="
echo ""
echo "总检查数: $TOTAL_CHECKS"
echo "通过数: $PASSED_CHECKS"
echo "失败数: $((TOTAL_CHECKS - PASSED_CHECKS))"
echo ""

if [ $PASSED_CHECKS -eq $TOTAL_CHECKS ]; then
    echo "✓ 所有接口响应字段符合规范"
else
    echo "✗ 部分接口响应字段不符合规范"
    echo ""
    echo "详细报告: $REPORT_FILE"
fi

cat "$REPORT_FILE"
