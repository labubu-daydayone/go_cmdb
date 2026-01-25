#!/bin/bash

# API Contract Test Script
# Tests that all APIs follow the standard response structure:
# - Success: { "code": 0, "message": "success", "data": {...} }
# - List: { "code": 0, "message": "success", "data": { "items": [], "total": 0, "page": 1, "pageSize": 20 } }
# - Error: { "code": != 0, "message": "...", "data": null }

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test result
test_result() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ $1 -eq 0 ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✓${NC} $2"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}✗${NC} $2"
        if [ -n "$3" ]; then
            echo -e "  ${YELLOW}Details:${NC} $3"
        fi
    fi
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is not installed. Please install jq first.${NC}"
    exit 1
fi

# Check field exists in JSON
check_field() {
    local json="$1"
    local field="$2"
    echo "$json" | jq -e "has(\"$field\")" > /dev/null 2>&1
}

# Check field value
check_field_value() {
    local json="$1"
    local field="$2"
    local expected="$3"
    local actual=$(echo "$json" | jq -r ".$field")
    [ "$actual" = "$expected" ]
}

# Test list API response structure
test_list_api() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    
    echo -e "\n${YELLOW}Testing List API:${NC} $name"
    
    # Make request
    if [ "$method" = "GET" ]; then
        response=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE_URL$url")
    else
        response=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "$API_BASE_URL$url")
    fi
    
    # Check code field
    if check_field "$response" "code"; then
        test_result 0 "Has 'code' field"
    else
        test_result 1 "Has 'code' field" "Response: $response"
        return
    fi
    
    # Check message field
    if check_field "$response" "message"; then
        test_result 0 "Has 'message' field"
    else
        test_result 1 "Has 'message' field"
    fi
    
    # Check data field
    if check_field "$response" "data"; then
        test_result 0 "Has 'data' field"
    else
        test_result 1 "Has 'data' field"
        return
    fi
    
    # Check data.items field
    if echo "$response" | jq -e ".data.items" > /dev/null 2>&1; then
        test_result 0 "Has 'data.items' field"
    else
        test_result 1 "Has 'data.items' field" "Found: $(echo "$response" | jq '.data | keys')"
    fi
    
    # Check data.total field
    if echo "$response" | jq -e ".data.total" > /dev/null 2>&1; then
        test_result 0 "Has 'data.total' field"
    else
        test_result 1 "Has 'data.total' field"
    fi
    
    # Check data.page field
    if echo "$response" | jq -e ".data.page" > /dev/null 2>&1; then
        test_result 0 "Has 'data.page' field"
    else
        test_result 1 "Has 'data.page' field"
    fi
    
    # Check data.pageSize field
    if echo "$response" | jq -e ".data.pageSize" > /dev/null 2>&1; then
        test_result 0 "Has 'data.pageSize' field"
    else
        test_result 1 "Has 'data.pageSize' field"
    fi
    
    # Check no 'list' field (should use 'items' instead)
    if echo "$response" | jq -e ".data.list" > /dev/null 2>&1; then
        test_result 1 "Should NOT have 'data.list' field (use 'items' instead)"
    else
        test_result 0 "Does not have 'data.list' field (correct)"
    fi
}

# Test detail API response structure
test_detail_api() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    
    echo -e "\n${YELLOW}Testing Detail API:${NC} $name"
    
    # Make request
    if [ "$method" = "GET" ]; then
        response=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE_URL$url")
    else
        response=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "$API_BASE_URL$url")
    fi
    
    # Check code field
    if check_field "$response" "code"; then
        test_result 0 "Has 'code' field"
    else
        test_result 1 "Has 'code' field" "Response: $response"
        return
    fi
    
    # Check message field
    if check_field "$response" "message"; then
        test_result 0 "Has 'message' field"
    else
        test_result 1 "Has 'message' field"
    fi
    
    # Check data field
    if check_field "$response" "data"; then
        test_result 0 "Has 'data' field"
    else
        test_result 1 "Has 'data' field"
    fi
    
    # Check data is object (not array)
    data_type=$(echo "$response" | jq -r '.data | type')
    if [ "$data_type" = "object" ] || [ "$data_type" = "null" ]; then
        test_result 0 "data is object or null (not array)"
    else
        test_result 1 "data should be object or null" "Found: $data_type"
    fi
}

# Test write API response structure
test_write_api() {
    local name="$1"
    local url="$2"
    local data="$3"
    
    echo -e "\n${YELLOW}Testing Write API:${NC} $name"
    
    # Make request
    response=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data" "$API_BASE_URL$url")
    
    # Check code field
    if check_field "$response" "code"; then
        test_result 0 "Has 'code' field"
    else
        test_result 1 "Has 'code' field" "Response: $response"
        return
    fi
    
    # Check message field
    if check_field "$response" "message"; then
        test_result 0 "Has 'message' field"
    else
        test_result 1 "Has 'message' field"
    fi
    
    # Check data field
    if check_field "$response" "data"; then
        test_result 0 "Has 'data' field"
    else
        test_result 1 "Has 'data' field"
    fi
}

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}API Contract Test${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "API Base URL: $API_BASE_URL"
echo -e "Token: ${TOKEN:0:20}..."

# Test List APIs
test_list_api "DNS Records List" "/api/v1/dns/records?page=1&pageSize=20"
test_list_api "ACME Requests List" "/api/v1/acme/certificate/requests?page=1&pageSize=20"
test_list_api "Agent Identities List" "/api/v1/agent/identities?page=1&pageSize=20"
test_list_api "Agent Tasks List" "/api/v1/agent/tasks?page=1&pageSize=20"
test_list_api "Config Versions List" "/api/v1/config/versions?page=1&pageSize=20"

# Test Detail APIs
test_detail_api "Health Check" "/api/v1/health"
test_detail_api "DNS Record Detail" "/api/v1/dns/records/1"
test_detail_api "ACME Request Detail" "/api/v1/acme/certificate/requests/1"

# Summary
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Test Summary${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
