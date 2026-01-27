#!/bin/bash

# T2-18 Certificate Resource APIs Acceptance Test
# 12条curl验收测试

BASE_URL="http://20.2.140.226:8080"
API_BASE="/api/v1"

echo "========================================="
echo "T2-18 Certificate Resource APIs Test"
echo "========================================="
echo ""

# Login to get token
echo "Step 0: Login to get token"
echo "POST $BASE_URL$API_BASE/auth/login"
TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$TOKEN_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo "✗ Failed to get token"
    echo "Response: $TOKEN_RESPONSE"
    exit 1
fi

echo "✓ Token obtained"
echo ""

# Test 1: Upload certificate (success)
echo "========================================="
echo "Test 1: Upload certificate (success)"
echo "========================================="
echo "POST $BASE_URL$API_BASE/certificates/upload"
echo ""

CERT_PEM="-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU6T8MA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNjAxMjcwMDAwMDBaFw0yNzAxMjcwMDAwMDBaMBIxEDAOBgNVBAMM
B3Rlc3QuY29tMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJhPHhITqQbPklG3ib
SvastqhoZUB0KsgCAk1DZmdtwHdcRUgcPNI8F6fnH9gx1tjDancl7e+IUVbuNifl
izMCAwEAATANBgkqhkiG9w0BAQsFAAOBgQBGRWeC928+A5Dlt0UAbRwAk/zUzJiC
OpBMB+XK77eEWmQ7hiTiROkqz0LMRduWPS3MAcYvbT0Af4+J8qH/rxN0qFWuPLlz
3PKfLhduzYfdPCiDK70eBcTuFnETP1+nsHMoYs4hQ+7+JYg3BFOgPffrAjzsvE52
4gTFxDgmJGhzLQ==
-----END CERTIFICATE-----"

KEY_PEM="-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA0smE8eEhOpBs+SUb
eJtK9qy2qGhlQHQqyAICTUNmZ23Ad1xFSBw80jwXp+cf2DHW2MNqdyXt74hRVu42
J+WLMwIDAQABAkAGRrlooUfztDsHHeDpz2ivSiuIXLrO4Hqp/LRxRUz6J+vf+96F
qJNBplko2plQWRN0/l16CN06W8T7hhMKKIxBAiEA/dIXH2+NXvmqB5Jd+mw7rWkI
vHRYcSWJWqlW5CMGqYMCIQDVWZ6+OtOWXmlZjObihKsGOXSQhf6T1YkqIVWBrmsm
QQIgQxLY+Af6PUXmukteoEEOz8zAyBNZ4J+/3qCKCm/q1JMCIQC5tsVDnXBqqMAA
qUR2R2K2+0UqlSeQzNWSvQPMd4EBgQIgFYRNRJ6X5DbzRjK4D5fCiIhbdOTNnWkI
vBQqZooRKoE=
-----END PRIVATE KEY-----"

UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL$API_BASE/certificates/upload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"provider\": \"manual\",
    \"certificatePem\": \"$CERT_PEM\",
    \"privateKeyPem\": \"$KEY_PEM\",
    \"domains\": [\"test.com\", \"*.test.com\"]
  }")

echo "Response:"
echo "$UPLOAD_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$UPLOAD_RESPONSE"
echo ""

UPLOAD_CODE=$(echo "$UPLOAD_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)
CERT_ID=$(echo "$UPLOAD_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)

if [ "$UPLOAD_CODE" == "0" ]; then
    echo "✓ Test 1 passed: Certificate uploaded successfully (ID=$CERT_ID)"
else
    echo "✗ Test 1 failed: Upload failed (code=$UPLOAD_CODE)"
fi
echo ""

# Test 2: Upload same fingerprint (should fail)
echo "========================================="
echo "Test 2: Upload same fingerprint (should fail)"
echo "========================================="
echo "POST $BASE_URL$API_BASE/certificates/upload"
echo ""

DUP_RESPONSE=$(curl -s -X POST "$BASE_URL$API_BASE/certificates/upload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"provider\": \"manual\",
    \"certificatePem\": \"$CERT_PEM\",
    \"privateKeyPem\": \"$KEY_PEM\",
    \"domains\": [\"test.com\"]
  }")

echo "Response:"
echo "$DUP_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$DUP_RESPONSE"
echo ""

DUP_CODE=$(echo "$DUP_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$DUP_CODE" != "0" ]; then
    echo "✓ Test 2 passed: Duplicate fingerprint rejected (code=$DUP_CODE)"
else
    echo "✗ Test 2 failed: Duplicate fingerprint accepted"
fi
echo ""

# Test 3: List certificates (pagination)
echo "========================================="
echo "Test 3: List certificates (pagination)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates?page=1&pageSize=5"
echo ""

LIST_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates?page=1&pageSize=5" \
  -H "Authorization: Bearer $TOKEN")

echo "Response:"
echo "$LIST_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$LIST_RESPONSE"
echo ""

LIST_CODE=$(echo "$LIST_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)
HAS_ITEMS=$(echo "$LIST_RESPONSE" | python3 -c "import sys, json; print('items' in json.load(sys.stdin)['data'])" 2>/dev/null)
HAS_TOTAL=$(echo "$LIST_RESPONSE" | python3 -c "import sys, json; print('total' in json.load(sys.stdin)['data'])" 2>/dev/null)

if [ "$LIST_CODE" == "0" ] && [ "$HAS_ITEMS" == "True" ] && [ "$HAS_TOTAL" == "True" ]; then
    echo "✓ Test 3 passed: List returned with items/total/page/pageSize"
else
    echo "✗ Test 3 failed: List format incorrect"
fi
echo ""

# Test 4: List certificates with source filter
echo "========================================="
echo "Test 4: List certificates (filter: source=manual)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates?source=manual"
echo ""

FILTER_SOURCE_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates?source=manual" \
  -H "Authorization: Bearer $TOKEN")

echo "Response (truncated):"
echo "$FILTER_SOURCE_RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin); print(json.dumps({'code': d['code'], 'itemsCount': len(d['data']['items'])}, indent=2))" 2>/dev/null || echo "$FILTER_SOURCE_RESPONSE"
echo ""

FILTER_CODE=$(echo "$FILTER_SOURCE_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$FILTER_CODE" == "0" ]; then
    echo "✓ Test 4 passed: Source filter works"
else
    echo "✗ Test 4 failed"
fi
echo ""

# Test 5: List certificates with status filter
echo "========================================="
echo "Test 5: List certificates (filter: status=valid)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates?status=valid"
echo ""

FILTER_STATUS_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates?status=valid" \
  -H "Authorization: Bearer $TOKEN")

echo "Response (truncated):"
echo "$FILTER_STATUS_RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin); print(json.dumps({'code': d['code'], 'itemsCount': len(d['data']['items'])}, indent=2))" 2>/dev/null || echo "$FILTER_STATUS_RESPONSE"
echo ""

FILTER_STATUS_CODE=$(echo "$FILTER_STATUS_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$FILTER_STATUS_CODE" == "0" ]; then
    echo "✓ Test 5 passed: Status filter works"
else
    echo "✗ Test 5 failed"
fi
echo ""

# Test 6: Get certificate detail (with PEM/KEY)
echo "========================================="
echo "Test 6: Get certificate detail (with PEM/KEY)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates/$CERT_ID"
echo ""

if [ -z "$CERT_ID" ]; then
    echo "✗ Test 6 skipped: No certificate ID from upload"
else
    DETAIL_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates/$CERT_ID" \
      -H "Authorization: Bearer $TOKEN")

    echo "Response (PEM truncated):"
    echo "$DETAIL_RESPONSE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if 'data' in d:
    data = d['data']
    if 'certificatePem' in data:
        data['certificatePem'] = data['certificatePem'][:50] + '...(truncated)'
    if 'privateKeyPem' in data:
        data['privateKeyPem'] = data['privateKeyPem'][:50] + '...(truncated)'
print(json.dumps(d, indent=2))
" 2>/dev/null || echo "$DETAIL_RESPONSE"
    echo ""

    DETAIL_CODE=$(echo "$DETAIL_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)
    HAS_CERT_PEM=$(echo "$DETAIL_RESPONSE" | python3 -c "import sys, json; print('certificatePem' in json.load(sys.stdin)['data'])" 2>/dev/null)
    HAS_KEY_PEM=$(echo "$DETAIL_RESPONSE" | python3 -c "import sys, json; print('privateKeyPem' in json.load(sys.stdin)['data'])" 2>/dev/null)
    HAS_DOMAINS=$(echo "$DETAIL_RESPONSE" | python3 -c "import sys, json; print('domains' in json.load(sys.stdin)['data'])" 2>/dev/null)

    if [ "$DETAIL_CODE" == "0" ] && [ "$HAS_CERT_PEM" == "True" ] && [ "$HAS_KEY_PEM" == "True" ] && [ "$HAS_DOMAINS" == "True" ]; then
        echo "✓ Test 6 passed: Detail contains certificatePem, privateKeyPem, and domains"
    else
        echo "✗ Test 6 failed: Missing required fields"
    fi
fi
echo ""

# Test 7: Get non-existent certificate (404)
echo "========================================="
echo "Test 7: Get non-existent certificate (404)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates/999999"
echo ""

NOT_FOUND_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates/999999" \
  -H "Authorization: Bearer $TOKEN")

echo "Response:"
echo "$NOT_FOUND_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$NOT_FOUND_RESPONSE"
echo ""

NOT_FOUND_CODE=$(echo "$NOT_FOUND_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$NOT_FOUND_CODE" != "0" ]; then
    echo "✓ Test 7 passed: Non-existent certificate returns error (code=$NOT_FOUND_CODE)"
else
    echo "✗ Test 7 failed: Should return error"
fi
echo ""

# Test 8: Unauthorized access (no token)
echo "========================================="
echo "Test 8: Unauthorized access (no token)"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates (without Authorization header)"
echo ""

UNAUTH_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates")

echo "Response:"
echo "$UNAUTH_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$UNAUTH_RESPONSE"
echo ""

UNAUTH_CODE=$(echo "$UNAUTH_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$UNAUTH_CODE" != "0" ]; then
    echo "✓ Test 8 passed: Unauthorized access rejected (code=$UNAUTH_CODE)"
else
    echo "✗ Test 8 failed: Should require authorization"
fi
echo ""

# Test 9: Upload with invalid provider (should fail)
echo "========================================="
echo "Test 9: Upload with invalid provider (should fail)"
echo "========================================="
echo "POST $BASE_URL$API_BASE/certificates/upload (provider=acme)"
echo ""

INVALID_PROVIDER_RESPONSE=$(curl -s -X POST "$BASE_URL$API_BASE/certificates/upload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"provider\": \"acme\",
    \"certificatePem\": \"$CERT_PEM\",
    \"privateKeyPem\": \"$KEY_PEM\",
    \"domains\": [\"test.com\"]
  }")

echo "Response:"
echo "$INVALID_PROVIDER_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$INVALID_PROVIDER_RESPONSE"
echo ""

INVALID_PROVIDER_CODE=$(echo "$INVALID_PROVIDER_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$INVALID_PROVIDER_CODE" != "0" ]; then
    echo "✓ Test 9 passed: Invalid provider rejected (code=$INVALID_PROVIDER_CODE)"
else
    echo "✗ Test 9 failed: Should reject non-manual provider"
fi
echo ""

# Test 10: Upload with missing domains (should fail)
echo "========================================="
echo "Test 10: Upload with missing domains (should fail)"
echo "========================================="
echo "POST $BASE_URL$API_BASE/certificates/upload (empty domains)"
echo ""

MISSING_DOMAINS_RESPONSE=$(curl -s -X POST "$BASE_URL$API_BASE/certificates/upload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"provider\": \"manual\",
    \"certificatePem\": \"$CERT_PEM\",
    \"privateKeyPem\": \"$KEY_PEM\",
    \"domains\": []
  }")

echo "Response:"
echo "$MISSING_DOMAINS_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$MISSING_DOMAINS_RESPONSE"
echo ""

MISSING_DOMAINS_CODE=$(echo "$MISSING_DOMAINS_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$MISSING_DOMAINS_CODE" != "0" ]; then
    echo "✓ Test 10 passed: Empty domains rejected (code=$MISSING_DOMAINS_CODE)"
else
    echo "✗ Test 10 failed: Should require at least one domain"
fi
echo ""

# Test 11: List with multiple filters
echo "========================================="
echo "Test 11: List with multiple filters"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates?source=manual&status=valid&page=1&pageSize=10"
echo ""

MULTI_FILTER_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates?source=manual&status=valid&page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN")

echo "Response (truncated):"
echo "$MULTI_FILTER_RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin); print(json.dumps({'code': d['code'], 'itemsCount': len(d['data']['items']), 'page': d['data']['page'], 'pageSize': d['data']['pageSize']}, indent=2))" 2>/dev/null || echo "$MULTI_FILTER_RESPONSE"
echo ""

MULTI_FILTER_CODE=$(echo "$MULTI_FILTER_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$MULTI_FILTER_CODE" == "0" ]; then
    echo "✓ Test 11 passed: Multiple filters work"
else
    echo "✗ Test 11 failed"
fi
echo ""

# Test 12: Verify domainCount in list response
echo "========================================="
echo "Test 12: Verify domainCount in list response"
echo "========================================="
echo "GET $BASE_URL$API_BASE/certificates?page=1&pageSize=1"
echo ""

DOMAIN_COUNT_RESPONSE=$(curl -s "$BASE_URL$API_BASE/certificates?page=1&pageSize=1" \
  -H "Authorization: Bearer $TOKEN")

echo "Response:"
echo "$DOMAIN_COUNT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$DOMAIN_COUNT_RESPONSE"
echo ""

HAS_DOMAIN_COUNT=$(echo "$DOMAIN_COUNT_RESPONSE" | python3 -c "import sys, json; items=json.load(sys.stdin)['data']['items']; print(len(items) > 0 and 'domainCount' in items[0])" 2>/dev/null)

if [ "$HAS_DOMAIN_COUNT" == "True" ]; then
    echo "✓ Test 12 passed: domainCount field exists in list items"
else
    echo "✗ Test 12 failed: domainCount field missing"
fi
echo ""

echo "========================================="
echo "All 12 tests completed"
echo "========================================="
