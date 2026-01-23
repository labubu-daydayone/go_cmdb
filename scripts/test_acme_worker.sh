#!/bin/bash

# T2-05 ACME Worker验收测试脚本
# 测试场景：单域名、wildcard、SAN、多网站共享、失败重试、Google EAB、challenge清理、自动apply_config

set -e

BASE_URL="http://localhost:3000/api/v1"
TOKEN="your_auth_token_here"

echo "========================================="
echo "T2-05 ACME Worker 验收测试"
echo "========================================="
echo ""

# 辅助函数
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -X $method "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json"
    else
        curl -s -X $method "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

echo "=== 1. 创建Let's Encrypt账户 ==="
LETSENCRYPT_ACCOUNT=$(api_call POST "/acme/account/create" '{
  "providerName": "letsencrypt",
  "email": "admin@example.com"
}')
echo "$LETSENCRYPT_ACCOUNT" | jq '.'
LETSENCRYPT_ACCOUNT_ID=$(echo "$LETSENCRYPT_ACCOUNT" | jq -r '.data.id')
echo "Let's Encrypt Account ID: $LETSENCRYPT_ACCOUNT_ID"
echo ""

echo "=== 2. 创建Google Public CA账户（需要EAB） ==="
GOOGLE_ACCOUNT=$(api_call POST "/acme/account/create" '{
  "providerName": "google",
  "email": "admin@example.com",
  "eabKid": "your_eab_kid_here",
  "eabHmacKey": "your_eab_hmac_key_here"
}')
echo "$GOOGLE_ACCOUNT" | jq '.'
GOOGLE_ACCOUNT_ID=$(echo "$GOOGLE_ACCOUNT" | jq -r '.data.id')
echo "Google Account ID: $GOOGLE_ACCOUNT_ID"
echo ""

echo "=== 3. 请求单域名证书（example.com） ==="
SINGLE_DOMAIN_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$LETSENCRYPT_ACCOUNT_ID',
  "domains": ["example.com"]
}')
echo "$SINGLE_DOMAIN_REQUEST" | jq '.'
SINGLE_DOMAIN_REQUEST_ID=$(echo "$SINGLE_DOMAIN_REQUEST" | jq -r '.data.id')
echo "Single Domain Request ID: $SINGLE_DOMAIN_REQUEST_ID"
echo ""

echo "=== 4. 请求wildcard证书（*.example.com） ==="
WILDCARD_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$LETSENCRYPT_ACCOUNT_ID',
  "domains": ["*.example.com"]
}')
echo "$WILDCARD_REQUEST" | jq '.'
WILDCARD_REQUEST_ID=$(echo "$WILDCARD_REQUEST" | jq -r '.data.id')
echo "Wildcard Request ID: $WILDCARD_REQUEST_ID"
echo ""

echo "=== 5. 请求SAN证书（example.com + www.example.com） ==="
SAN_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$LETSENCRYPT_ACCOUNT_ID',
  "domains": ["example.com", "www.example.com"]
}')
echo "$SAN_REQUEST" | jq '.'
SAN_REQUEST_ID=$(echo "$SAN_REQUEST" | jq -r '.data.id')
echo "SAN Request ID: $SAN_REQUEST_ID"
echo ""

echo "=== 6. 请求多域名SAN证书（example.com + *.example.com + www.example.com） ==="
MULTI_SAN_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$LETSENCRYPT_ACCOUNT_ID',
  "domains": ["example.com", "*.example.com", "www.example.com"]
}')
echo "$MULTI_SAN_REQUEST" | jq '.'
MULTI_SAN_REQUEST_ID=$(echo "$MULTI_SAN_REQUEST" | jq -r '.data.id')
echo "Multi SAN Request ID: $MULTI_SAN_REQUEST_ID"
echo ""

echo "=== 7. 使用Google Public CA请求证书 ==="
GOOGLE_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$GOOGLE_ACCOUNT_ID',
  "domains": ["test.example.com"]
}')
echo "$GOOGLE_REQUEST" | jq '.'
GOOGLE_REQUEST_ID=$(echo "$GOOGLE_REQUEST" | jq -r '.data.id')
echo "Google Request ID: $GOOGLE_REQUEST_ID"
echo ""

echo "=== 8. 查询所有证书请求 ==="
api_call GET "/acme/certificate/requests" | jq '.'
echo ""

echo "=== 9. 查询pending状态的证书请求 ==="
api_call GET "/acme/certificate/requests?status=pending" | jq '.'
echo ""

echo "=== 10. 查询单条证书请求详情 ==="
api_call GET "/acme/certificate/requests/$SINGLE_DOMAIN_REQUEST_ID" | jq '.'
echo ""

echo "=== 11. 等待ACME Worker处理（40秒） ==="
echo "等待中..."
sleep 40
echo ""

echo "=== 12. 再次查询证书请求状态（应该变为running或success） ==="
api_call GET "/acme/certificate/requests/$SINGLE_DOMAIN_REQUEST_ID" | jq '.'
echo ""

echo "=== 13. 查询所有success状态的证书请求 ==="
api_call GET "/acme/certificate/requests?status=success" | jq '.'
echo ""

echo "=== 14. 查询所有failed状态的证书请求 ==="
api_call GET "/acme/certificate/requests?status=failed" | jq '.'
echo ""

echo "=== 15. 构造一个失败场景（错误的domain） ==="
FAILED_REQUEST=$(api_call POST "/acme/certificate/request" '{
  "accountId": '$LETSENCRYPT_ACCOUNT_ID',
  "domains": ["invalid-domain-that-does-not-exist-12345.com"]
}')
echo "$FAILED_REQUEST" | jq '.'
FAILED_REQUEST_ID=$(echo "$FAILED_REQUEST" | jq -r '.data.id')
echo "Failed Request ID: $FAILED_REQUEST_ID"
echo ""

echo "=== 16. 等待失败请求处理（40秒） ==="
echo "等待中..."
sleep 40
echo ""

echo "=== 17. 查询失败请求详情（应该有last_error） ==="
api_call GET "/acme/certificate/requests/$FAILED_REQUEST_ID" | jq '.'
echo ""

echo "=== 18. 手动重试失败请求 ==="
api_call POST "/acme/certificate/retry" '{
  "id": '$FAILED_REQUEST_ID'
}' | jq '.'
echo ""

echo "=== 19. 查询重试后的请求状态（应该变为pending） ==="
api_call GET "/acme/certificate/requests/$FAILED_REQUEST_ID" | jq '.'
echo ""

echo "=== 20. 查询按accountId筛选的证书请求 ==="
api_call GET "/acme/certificate/requests?accountId=$LETSENCRYPT_ACCOUNT_ID" | jq '.'
echo ""

echo "=== 21. 查询分页结果（page=1, pageSize=5） ==="
api_call GET "/acme/certificate/requests?page=1&pageSize=5" | jq '.'
echo ""

echo "========================================="
echo "SQL验收测试（15条）"
echo "========================================="
echo ""

echo "=== SQL 1: 查询所有pending状态的certificate_requests ==="
echo "SELECT id, account_id, domains, status, attempts, created_at FROM certificate_requests WHERE status = 'pending';"
echo ""

echo "=== SQL 2: 查询所有success状态的certificate_requests ==="
echo "SELECT id, account_id, domains, status, result_certificate_id, created_at FROM certificate_requests WHERE status = 'success';"
echo ""

echo "=== SQL 3: 查询所有failed状态的certificate_requests（含错误信息） ==="
echo "SELECT id, account_id, domains, status, attempts, last_error, created_at FROM certificate_requests WHERE status = 'failed';"
echo ""

echo "=== SQL 4: 按status统计certificate_requests数量 ==="
echo "SELECT status, COUNT(*) as count FROM certificate_requests GROUP BY status;"
echo ""

echo "=== SQL 5: 查询attempts >= 5的certificate_requests ==="
echo "SELECT id, account_id, domains, status, attempts, last_error FROM certificate_requests WHERE attempts >= 5;"
echo ""

echo "=== SQL 6: 查询所有已签发的certificates（含fingerprint） ==="
echo "SELECT id, name, fingerprint, status, issuer, expires_at, created_at FROM certificates WHERE status = 'issued';"
echo ""

echo "=== SQL 7: 查询所有certificate_domains（SAN） ==="
echo "SELECT id, certificate_id, domain, is_wildcard, created_at FROM certificate_domains ORDER BY certificate_id, domain;"
echo ""

echo "=== SQL 8: 按certificate_id统计certificate_domains数量 ==="
echo "SELECT certificate_id, COUNT(*) as domain_count FROM certificate_domains GROUP BY certificate_id;"
echo ""

echo "=== SQL 9: 查询wildcard证书域名 ==="
echo "SELECT cd.id, cd.certificate_id, cd.domain, c.name, c.fingerprint FROM certificate_domains cd JOIN certificates c ON cd.certificate_id = c.id WHERE cd.is_wildcard = TRUE;"
echo ""

echo "=== SQL 10: 查询所有ACME账户 ==="
echo "SELECT id, provider_id, email, status, registration_uri, created_at FROM acme_accounts;"
echo ""

echo "=== SQL 11: 查询需要EAB的ACME providers ==="
echo "SELECT id, name, directory_url, requires_eab, status FROM acme_providers WHERE requires_eab = TRUE;"
echo ""

echo "=== SQL 12: 关联查询certificate_requests和certificates ==="
echo "SELECT cr.id as request_id, cr.domains, cr.status as request_status, c.id as cert_id, c.fingerprint, c.issuer FROM certificate_requests cr LEFT JOIN certificates c ON cr.result_certificate_id = c.id;"
echo ""

echo "=== SQL 13: 查询ACME challenge TXT记录（owner_type=acme_challenge） ==="
echo "SELECT id, domain_id, type, name, value, status, desired_state, owner_type, owner_id, created_at FROM domain_dns_records WHERE owner_type = 'acme_challenge';"
echo ""

echo "=== SQL 14: 查询已删除的ACME challenge记录（desired_state=absent） ==="
echo "SELECT id, domain_id, type, name, value, status, desired_state, owner_type, owner_id, created_at FROM domain_dns_records WHERE owner_type = 'acme_challenge' AND desired_state = 'absent';"
echo ""

echo "=== SQL 15: 查询certificate_bindings（证书与网站绑定关系） ==="
echo "SELECT id, certificate_id, website_id, status, created_at FROM certificate_bindings;"
echo ""

echo "========================================="
echo "验收测试完成"
echo "========================================="
echo ""

echo "验收要点："
echo "1. 单域名/wildcard/SAN证书请求成功创建"
echo "2. Google Public CA EAB账户创建成功"
echo "3. 证书请求状态流转：pending → running → success/failed"
echo "4. 失败请求有last_error和attempts递增"
echo "5. 手动重试功能正常"
echo "6. certificate_domains表记录所有SAN域名"
echo "7. certificates表fingerprint唯一"
echo "8. ACME challenge TXT记录创建和清理"
echo "9. certificate_bindings关联证书和网站"
echo "10. API分页和筛选功能正常"
