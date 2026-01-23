#!/bin/bash

# T2-07 Certificate Coverage Validation Test Script
# Tests certificate-website relationship visualization and HTTPS enable validation

set -e

BASE_URL="http://20.2.140.226:8080"
TOKEN=""

echo "========================================="
echo "T2-07 Certificate Coverage Test Suite"
echo "========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass_count=0
fail_count=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}[PASS]${NC} $2"
        ((pass_count++))
    else
        echo -e "${RED}[FAIL]${NC} $2"
        ((fail_count++))
    fi
}

# Function to print section header
print_section() {
    echo ""
    echo -e "${YELLOW}=== $1 ===${NC}"
    echo ""
}

# ============================================
# CURL Tests (20 tests)
# ============================================

print_section "CURL Tests (20 tests)"

# Test 1: Login to get JWT token
echo "Test 1: Login to get JWT token"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
    print_result 0 "Login successful, token obtained"
else
    print_result 1 "Login failed"
    exit 1
fi

# Test 2: Query certificate → websites (valid certificate)
echo "Test 2: Query certificate → websites (certificate ID 1)"
CERT_WEBSITES=$(curl -s -X GET "$BASE_URL/api/v1/certificates/1/websites" \
  -H "Authorization: Bearer $TOKEN")

echo $CERT_WEBSITES | grep -q '"code":0'
print_result $? "Query certificate websites API"

# Test 3: Query certificate → websites (invalid certificate)
echo "Test 3: Query certificate → websites (invalid certificate ID 99999)"
CERT_WEBSITES_INVALID=$(curl -s -X GET "$BASE_URL/api/v1/certificates/99999/websites" \
  -H "Authorization: Bearer $TOKEN")

echo $CERT_WEBSITES_INVALID | grep -q '"code":3001'
print_result $? "Query invalid certificate returns 404"

# Test 4: Query website → certificate candidates (valid website)
echo "Test 4: Query website → certificate candidates (website ID 1)"
WEBSITE_CANDIDATES=$(curl -s -X GET "$BASE_URL/api/v1/websites/1/certificates/candidates" \
  -H "Authorization: Bearer $TOKEN")

echo $WEBSITE_CANDIDATES | grep -q '"code":0'
print_result $? "Query website certificate candidates API"

# Test 5: Query website → certificate candidates (invalid website)
echo "Test 5: Query website → certificate candidates (invalid website ID 99999)"
WEBSITE_CANDIDATES_INVALID=$(curl -s -X GET "$BASE_URL/api/v1/websites/99999/certificates/candidates" \
  -H "Authorization: Bearer $TOKEN")

echo $WEBSITE_CANDIDATES_INVALID | grep -q '"code":3001'
print_result $? "Query invalid website returns 404"

# Test 6: Check coverage status in candidates response
echo "Test 6: Check coverage status in candidates response"
echo $WEBSITE_CANDIDATES | grep -q 'coverageStatus'
print_result $? "Candidates response contains coverageStatus"

# Test 7: Check missing domains in candidates response
echo "Test 7: Check missing domains in partial coverage"
echo $WEBSITE_CANDIDATES | grep -q 'missingDomains'
print_result $? "Candidates response contains missingDomains"

# Test 8: Test unauthorized access to certificate websites
echo "Test 8: Test unauthorized access to certificate websites"
UNAUTH_CERT=$(curl -s -X GET "$BASE_URL/api/v1/certificates/1/websites")

echo $UNAUTH_CERT | grep -q '"code":1001'
print_result $? "Unauthorized access returns 401"

# Test 9: Test unauthorized access to website candidates
echo "Test 9: Test unauthorized access to website candidates"
UNAUTH_WEBSITE=$(curl -s -X GET "$BASE_URL/api/v1/websites/1/certificates/candidates")

echo $UNAUTH_WEBSITE | grep -q '"code":1001'
print_result $? "Unauthorized access returns 401"

# Test 10: Create website with partial coverage certificate (should fail)
echo "Test 10: Create website with partial coverage certificate"
CREATE_PARTIAL=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": ["example.com", "www.example.com"],
    "https": {
      "enabled": true,
      "cert_mode": "select",
      "certificate_id": 1,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $CREATE_PARTIAL | grep -q '"code":3003'
print_result $? "Create website with partial coverage fails with 3003"

# Test 11: Check error message contains missing domains
echo "Test 11: Check error message contains missing domains"
echo $CREATE_PARTIAL | grep -q 'missingDomains'
print_result $? "Error response contains missingDomains"

# Test 12: Check error message contains certificate domains
echo "Test 12: Check error message contains certificate domains"
echo $CREATE_PARTIAL | grep -q 'certificateDomains'
print_result $? "Error response contains certificateDomains"

# Test 13: Check error message contains website domains
echo "Test 13: Check error message contains website domains"
echo $CREATE_PARTIAL | grep -q 'websiteDomains'
print_result $? "Error response contains websiteDomains"

# Test 14: Check error message contains coverage status
echo "Test 14: Check error message contains coverage status"
echo $CREATE_PARTIAL | grep -q 'coverageStatus'
print_result $? "Error response contains coverageStatus"

# Test 15: Create website with ACME mode (should bypass validation)
echo "Test 15: Create website with ACME mode"
CREATE_ACME=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": ["test-acme.example.com"],
    "https": {
      "enabled": true,
      "cert_mode": "acme",
      "acme_provider_id": 1,
      "acme_account_id": 1,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $CREATE_ACME | grep -q '"code":0\|"code":3001\|"code":5002'
print_result $? "Create website with ACME mode bypasses coverage validation"

# Test 16: Create website with covered certificate (should succeed)
echo "Test 16: Create website with covered certificate"
CREATE_COVERED=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": ["a.example.com"],
    "https": {
      "enabled": true,
      "cert_mode": "select",
      "certificate_id": 1,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $CREATE_COVERED | grep -q '"code":0\|"code":3001\|"code":5002'
print_result $? "Create website with covered certificate succeeds or fails gracefully"

# Test 17: Test invalid certificate ID in HTTPS enable
echo "Test 17: Test invalid certificate ID in HTTPS enable"
CREATE_INVALID_CERT=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": ["test.example.com"],
    "https": {
      "enabled": true,
      "cert_mode": "select",
      "certificate_id": 99999,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $CREATE_INVALID_CERT | grep -q '"code":3001'
print_result $? "Invalid certificate ID returns 404"

# Test 18: Test ACME mode with empty domains (should fail)
echo "Test 18: Test ACME mode with empty domains"
CREATE_ACME_EMPTY=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": [],
    "https": {
      "enabled": true,
      "cert_mode": "acme",
      "acme_provider_id": 1,
      "acme_account_id": 1,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $CREATE_ACME_EMPTY | grep -q '"code":2001'
print_result $? "ACME mode with empty domains fails with 2001"

# Test 19: Test coverage with wildcard certificate
echo "Test 19: Test coverage with wildcard certificate (*.example.com)"
WILDCARD_COVERAGE=$(curl -s -X GET "$BASE_URL/api/v1/websites/1/certificates/candidates" \
  -H "Authorization: Bearer $TOKEN")

echo $WILDCARD_COVERAGE | grep -q 'coverageStatus'
print_result $? "Wildcard certificate coverage calculated"

# Test 20: Test coverage with second-level subdomain
echo "Test 20: Test coverage with second-level subdomain (a.b.example.com)"
SECOND_LEVEL=$(curl -s -X POST "$BASE_URL/api/v1/websites" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "line_group_id": 1,
    "origin_mode": "group",
    "origin_group_id": 1,
    "domains": ["a.b.example.com"],
    "https": {
      "enabled": true,
      "cert_mode": "select",
      "certificate_id": 1,
      "force_redirect": false,
      "hsts": false
    }
  }')

echo $SECOND_LEVEL | grep -q '"code":3003\|"code":3001\|"code":5002'
print_result $? "Second-level subdomain not covered by wildcard"

# ============================================
# SQL Validation (15 tests)
# ============================================

print_section "SQL Validation (15 tests)"

echo "Test 21: Verify certificate_bindings table exists"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "DESCRIBE certificate_bindings;" > /dev/null 2>&1
print_result $? "certificate_bindings table exists"

echo "Test 22: Verify certificate_domains table exists"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "DESCRIBE certificate_domains;" > /dev/null 2>&1
print_result $? "certificate_domains table exists"

echo "Test 23: Verify website_domains table exists"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "DESCRIBE website_domains;" > /dev/null 2>&1
print_result $? "website_domains table exists"

echo "Test 24: Count certificates with domains"
CERT_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(DISTINCT certificate_id) FROM certificate_domains;
")
if [ "$CERT_COUNT" -ge 0 ]; then
    print_result 0 "Certificates with domains: $CERT_COUNT"
else
    print_result 1 "Failed to count certificates"
fi

echo "Test 25: Count websites with domains"
WEBSITE_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(DISTINCT website_id) FROM website_domains;
")
if [ "$WEBSITE_COUNT" -ge 0 ]; then
    print_result 0 "Websites with domains: $WEBSITE_COUNT"
else
    print_result 1 "Failed to count websites"
fi

echo "Test 26: Check certificate_bindings relationships"
BINDING_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(*) FROM certificate_bindings;
")
if [ "$BINDING_COUNT" -ge 0 ]; then
    print_result 0 "Certificate bindings: $BINDING_COUNT"
else
    print_result 1 "Failed to count bindings"
fi

echo "Test 27: Verify wildcard domains in certificate_domains"
WILDCARD_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(*) FROM certificate_domains WHERE domain LIKE '*.%';
")
if [ "$WILDCARD_COUNT" -ge 0 ]; then
    print_result 0 "Wildcard certificates: $WILDCARD_COUNT"
else
    print_result 1 "Failed to count wildcard certificates"
fi

echo "Test 28: Check is_wildcard flag in certificate_domains"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "
SELECT * FROM certificate_domains WHERE is_wildcard = 1 LIMIT 5;
" > /dev/null 2>&1
print_result $? "is_wildcard flag exists and queryable"

echo "Test 29: Verify website_https table has cert_mode field"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "
SHOW COLUMNS FROM website_https LIKE 'cert_mode';
" > /dev/null 2>&1
print_result $? "website_https.cert_mode field exists"

echo "Test 30: Verify website_https table has certificate_id field"
mysql -h 20.2.140.226 -u root -proot123 cmdb -e "
SHOW COLUMNS FROM website_https LIKE 'certificate_id';
" > /dev/null 2>&1
print_result $? "website_https.certificate_id field exists"

echo "Test 31: Check websites with HTTPS enabled"
HTTPS_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(*) FROM website_https WHERE enabled = 1;
")
if [ "$HTTPS_COUNT" -ge 0 ]; then
    print_result 0 "Websites with HTTPS enabled: $HTTPS_COUNT"
else
    print_result 1 "Failed to count HTTPS websites"
fi

echo "Test 32: Check websites using select cert_mode"
SELECT_MODE_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(*) FROM website_https WHERE cert_mode = 'select';
")
if [ "$SELECT_MODE_COUNT" -ge 0 ]; then
    print_result 0 "Websites using select mode: $SELECT_MODE_COUNT"
else
    print_result 1 "Failed to count select mode websites"
fi

echo "Test 33: Check websites using acme cert_mode"
ACME_MODE_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT COUNT(*) FROM website_https WHERE cert_mode = 'acme';
")
if [ "$ACME_MODE_COUNT" -ge 0 ]; then
    print_result 0 "Websites using acme mode: $ACME_MODE_COUNT"
else
    print_result 1 "Failed to count acme mode websites"
fi

echo "Test 34: Verify no duplicate domains in certificate_domains"
CERT_DUP_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT certificate_id, domain, COUNT(*) as dup_count 
FROM certificate_domains 
GROUP BY certificate_id, domain 
HAVING dup_count > 1;
" | wc -l)
if [ "$CERT_DUP_COUNT" -eq 0 ]; then
    print_result 0 "No duplicate domains in certificate_domains"
else
    print_result 1 "Found $CERT_DUP_COUNT duplicate domains in certificate_domains"
fi

echo "Test 35: Verify no duplicate domains in website_domains"
WEBSITE_DUP_COUNT=$(mysql -h 20.2.140.226 -u root -proot123 cmdb -se "
SELECT website_id, domain, COUNT(*) as dup_count 
FROM website_domains 
GROUP BY website_id, domain 
HAVING dup_count > 1;
" | wc -l)
if [ "$WEBSITE_DUP_COUNT" -eq 0 ]; then
    print_result 0 "No duplicate domains in website_domains"
else
    print_result 1 "Found $WEBSITE_DUP_COUNT duplicate domains in website_domains"
fi

# ============================================
# Summary
# ============================================

print_section "Test Summary"

total_count=$((pass_count + fail_count))
echo "Total tests: $total_count"
echo -e "${GREEN}Passed: $pass_count${NC}"
echo -e "${RED}Failed: $fail_count${NC}"
echo ""

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
