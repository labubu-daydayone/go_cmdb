#!/bin/bash

# T2-17 Acceptance Test Script
# Test ACME set-default auto-activation for pending accounts

BASE_URL="http://20.2.140.226:8080"
API_BASE="${BASE_URL}/api/v1"

# Login to get token
echo "Login to get authentication token..."
TOKEN=$(curl -s -X POST "${API_BASE}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "ERROR: Failed to login and get token"
    exit 1
fi

echo "Token obtained: ${TOKEN:0:50}..."
echo ""

echo "========================================="
echo "T2-17 ACME Set-Default Auto-Activation"
echo "Acceptance Test"
echo "========================================="
echo ""

# Step 1: Create first account (should be pending)
echo "Step 1: Create first ACME account"
echo "POST ${API_BASE}/acme/account/create"
TIMESTAMP=$(date +%s)
EMAIL1="test-t2-17-${TIMESTAMP}-account1@example.com"
RESPONSE1=$(curl -s -X POST "${API_BASE}/acme/account/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"providerName\": \"letsencrypt\",
    \"email\": \"$EMAIL1\",
    \"eabKid\": \"\",
    \"eabHmacKey\": \"\"
  }")

echo "Response:"
echo "$RESPONSE1" | jq '.'
echo ""

# Extract account ID
ACCOUNT_ID_1=$(echo "$RESPONSE1" | jq -r '.data.items[0].id')
ACCOUNT_STATUS_1=$(echo "$RESPONSE1" | jq -r '.data.items[0].status')

if [ "$ACCOUNT_ID_1" == "null" ] || [ -z "$ACCOUNT_ID_1" ]; then
    echo "ERROR: Failed to create account 1"
    exit 1
fi

echo "Account 1 created: ID=$ACCOUNT_ID_1, Status=$ACCOUNT_STATUS_1"
echo ""

# Step 2: Set first account as default (核心验收：pending账号自动激活)
echo "Step 2: Set pending account as default (auto-activate)"
echo "POST ${API_BASE}/acme/accounts/set-default"
RESPONSE2=$(curl -s -X POST "${API_BASE}/acme/accounts/set-default" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"providerId\": 1,
    \"accountId\": $ACCOUNT_ID_1
  }")

echo "Response:"
echo "$RESPONSE2" | jq '.'
echo ""

# Check response
CODE2=$(echo "$RESPONSE2" | jq -r '.code')
if [ "$CODE2" != "0" ]; then
    echo "ERROR: Set-default failed with code=$CODE2"
    echo "Message: $(echo "$RESPONSE2" | jq -r '.message')"
    exit 1
fi

echo "✓ Set-default succeeded (code=0)"
echo ""

# Step 3: Query account list to verify status and default flag
echo "Step 3: Query account list to verify activation"
echo "GET ${API_BASE}/acme/accounts?page=1&pageSize=20"
RESPONSE3=$(curl -s "${API_BASE}/acme/accounts?page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN")

echo "Response:"
echo "$RESPONSE3" | jq '.'
echo ""

# Find the account in list
ACCOUNT_IN_LIST=$(echo "$RESPONSE3" | jq ".data.items[] | select(.id == $ACCOUNT_ID_1)")
ACCOUNT_STATUS_AFTER=$(echo "$ACCOUNT_IN_LIST" | jq -r '.status')
ACCOUNT_IS_DEFAULT=$(echo "$ACCOUNT_IN_LIST" | jq -r '.isDefault')

echo "Account 1 after set-default:"
echo "  Status: $ACCOUNT_STATUS_AFTER"
echo "  IsDefault: $ACCOUNT_IS_DEFAULT"
echo ""

# Verify status is active
if [ "$ACCOUNT_STATUS_AFTER" != "active" ]; then
    echo "ERROR: Account status is not 'active' (got: $ACCOUNT_STATUS_AFTER)"
    exit 1
fi

echo "✓ Account status is 'active'"

# Verify isDefault is true
if [ "$ACCOUNT_IS_DEFAULT" != "true" ]; then
    echo "ERROR: Account isDefault is not true (got: $ACCOUNT_IS_DEFAULT)"
    exit 1
fi

echo "✓ Account isDefault is true"
echo ""

# Step 4: Verify only one default account
DEFAULT_COUNT=$(echo "$RESPONSE3" | jq '[.data.items[] | select(.isDefault == true)] | length')
echo "Default accounts count: $DEFAULT_COUNT"

if [ "$DEFAULT_COUNT" != "1" ]; then
    echo "ERROR: Expected 1 default account, got $DEFAULT_COUNT"
    exit 1
fi

echo "✓ Only one default account exists"
echo ""

# Step 5: Create second account and set as default
echo "Step 5: Create second account"
EMAIL2="test-t2-17-${TIMESTAMP}-account2@example.com"
RESPONSE5=$(curl -s -X POST "${API_BASE}/acme/account/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"providerName\": \"letsencrypt\",
    \"email\": \"$EMAIL2\",
    \"eabKid\": \"\",
    \"eabHmacKey\": \"\"
  }")

echo "Response:"
echo "$RESPONSE5" | jq '.'
echo ""

ACCOUNT_ID_2=$(echo "$RESPONSE5" | jq -r '.data.items[0].id')

if [ "$ACCOUNT_ID_2" == "null" ] || [ -z "$ACCOUNT_ID_2" ]; then
    echo "ERROR: Failed to create account 2"
    exit 1
fi

echo "Account 2 created: ID=$ACCOUNT_ID_2"
echo ""

# Step 6: Set second account as default
echo "Step 6: Set second account as default"
RESPONSE6=$(curl -s -X POST "${API_BASE}/acme/accounts/set-default" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"providerId\": 1,
    \"accountId\": $ACCOUNT_ID_2
  }")

echo "Response:"
echo "$RESPONSE6" | jq '.'
echo ""

CODE6=$(echo "$RESPONSE6" | jq -r '.code')
if [ "$CODE6" != "0" ]; then
    echo "ERROR: Set-default for account 2 failed with code=$CODE6"
    exit 1
fi

echo "✓ Second account set as default"
echo ""

# Step 7: Verify first account is no longer default
echo "Step 7: Verify default switch"
RESPONSE7=$(curl -s "${API_BASE}/acme/accounts?page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN")

ACCOUNT1_IS_DEFAULT=$(echo "$RESPONSE7" | jq ".data.items[] | select(.id == $ACCOUNT_ID_1) | .isDefault")
ACCOUNT2_IS_DEFAULT=$(echo "$RESPONSE7" | jq ".data.items[] | select(.id == $ACCOUNT_ID_2) | .isDefault")

echo "Account 1 isDefault: $ACCOUNT1_IS_DEFAULT"
echo "Account 2 isDefault: $ACCOUNT2_IS_DEFAULT"
echo ""

if [ "$ACCOUNT1_IS_DEFAULT" != "false" ]; then
    echo "ERROR: Account 1 should not be default anymore"
    exit 1
fi

if [ "$ACCOUNT2_IS_DEFAULT" != "true" ]; then
    echo "ERROR: Account 2 should be default"
    exit 1
fi

echo "✓ Default switched correctly"
echo ""

# Verify only one default
DEFAULT_COUNT_FINAL=$(echo "$RESPONSE7" | jq '[.data.items[] | select(.isDefault == true)] | length')
if [ "$DEFAULT_COUNT_FINAL" != "1" ]; then
    echo "ERROR: Expected 1 default account, got $DEFAULT_COUNT_FINAL"
    exit 1
fi

echo "✓ Still only one default account"
echo ""

echo "========================================="
echo "ALL TESTS PASSED ✓"
echo "========================================="
echo ""
echo "Summary:"
echo "- Created pending account and set as default: SUCCESS"
echo "- Account auto-activated (pending → active): SUCCESS"
echo "- Default flag set correctly: SUCCESS"
echo "- Default uniqueness maintained: SUCCESS"
echo "- Default switch works correctly: SUCCESS"
