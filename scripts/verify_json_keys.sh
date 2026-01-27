#!/bin/bash
# 自动化验证所有 API 接口的 JSON 字段命名规范
# 用途：确保所有接口返回的字段都是 lowerCamelCase，没有 snake_case

set -e

API_BASE="${API_BASE:-http://20.2.140.226:8080}"
REPORT_FILE="/tmp/verify_json_keys_report.txt"

echo "========================================="
echo "API JSON 字段命名规范自动化验证"
echo "========================================="
echo ""
echo "API Base: $API_BASE"
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
check_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    
    echo -n "检查: $name ... "
    
    if [ "$method" = "GET" ]; then
        RESPONSE=$(curl -s -X GET "$API_BASE$endpoint" -H "Authorization: Bearer $TOKEN")
    else
        RESPONSE=$(curl -s -X POST "$API_BASE$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    # 检查响应是否包含 code 和 message（基本结构验证）
    if ! echo "$RESPONSE" | grep -q '"code"'; then
        echo "✗ 无效响应" | tee -a "$REPORT_FILE"
        echo "  端点: $method $endpoint" >> "$REPORT_FILE"
        echo "  响应: $RESPONSE" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        return 1
    fi
    
    # 检查是否包含 snake_case 字段（排除 code, message, data 这些顶层字段）
    SNAKE_FIELDS=$(echo "$RESPONSE" | grep -oE '"[a-z]+_[a-z_]+"' | grep -v '"code"' | grep -v '"message"' | grep -v '"data"' | sort -u || true)
    
    if [ -n "$SNAKE_FIELDS" ]; then
        echo "✗ 发现 snake_case 字段" | tee -a "$REPORT_FILE"
        echo "  端点: $method $endpoint" >> "$REPORT_FILE"
        echo "  不符合规范的字段:" >> "$REPORT_FILE"
        echo "$SNAKE_FIELDS" | while read -r field; do
            echo "    $field" >> "$REPORT_FILE"
        done
        echo "" >> "$REPORT_FILE"
        return 1
    else
        echo "✓ 通过"
        echo "  端点: $method $endpoint - ✓ 通过" >> "$REPORT_FILE"
        return 0
    fi
}

TOTAL_CHECKS=0
PASSED_CHECKS=0

echo "开始验证接口..."
echo ""

# P0-1: Nodes
echo "=== P0-1: Nodes ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "Nodes List" "GET" "/api/v1/nodes?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true
((TOTAL_CHECKS++)) && check_endpoint "Node Get" "GET" "/api/v1/nodes/7" "" && ((PASSED_CHECKS++)) || true

# P0-2: Domains
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-2: Domains ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "Domains List" "GET" "/api/v1/domains?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

# P0-2: DNS Records
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-2: DNS Records ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "DNS Records List" "GET" "/api/v1/dns/records?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

# P0-3: Certificates
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-3: Certificates ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "Certificates Lifecycle" "GET" "/api/v1/certificates?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

# P0-4: API Keys
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-4: API Keys ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "API Keys List" "GET" "/api/v1/api-keys?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

# P0-4: Websites
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-4: Websites ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "Websites List" "GET" "/api/v1/websites?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

# P0-4: Agent Tasks
echo "" | tee -a "$REPORT_FILE"
echo "=== P0-4: Agent Tasks ===" | tee -a "$REPORT_FILE"
((TOTAL_CHECKS++)) && check_endpoint "Agent Tasks List" "GET" "/api/v1/agent-tasks?page=1&pageSize=10" "" && ((PASSED_CHECKS++)) || true

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
    echo "✓ 所有接口响应字段符合 lowerCamelCase 规范"
    exit 0
else
    echo "✗ 部分接口响应字段不符合规范"
    echo ""
    echo "详细报告: $REPORT_FILE"
    exit 1
fi
