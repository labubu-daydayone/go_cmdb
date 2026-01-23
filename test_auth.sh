#!/bin/bash

# Test script for T0-03 auth implementation
# This script tests JWT authentication and authorization

set -e

echo "========================================="
echo "T0-03 Auth Test Suite"
echo "========================================="
echo ""

# Check if server is running
if ! curl -s http://localhost:8080/api/v1/ping > /dev/null 2>&1; then
    echo "Error: Server is not running on port 8080"
    echo "Please start the server first:"
    echo "  ./bin/cmdb"
    exit 1
fi

echo "Server is running. Starting tests..."
echo ""

# Test 1: Ping (no auth required)
echo "========================================="
echo "Test 1: GET /api/v1/ping (No Auth Required)"
echo "========================================="
echo ""
echo "Command:"
echo "curl http://localhost:8080/api/v1/ping"
echo ""
echo "Response:"
curl -s http://localhost:8080/api/v1/ping | jq .
echo ""
echo "Expected: code=0, message='success', data.pong=true"
echo ""

# Test 2: Login success
echo "========================================="
echo "Test 2: POST /api/v1/auth/login (Success)"
echo "========================================="
echo ""
echo "Command:"
echo 'curl -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '"'"'{"username":"admin","password":"admin123"}'"'"
echo ""
echo "Response:"
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
echo "$LOGIN_RESPONSE" | jq .
echo ""
echo "Expected: code=0, message='success', data.token exists"
echo ""

# Extract token
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "Warning: Failed to extract token, some tests may fail"
    TOKEN=""
fi

# Test 3: Login failure (wrong password)
echo "========================================="
echo "Test 3: POST /api/v1/auth/login (Wrong Password)"
echo "========================================="
echo ""
echo "Command:"
echo 'curl -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '"'"'{"username":"admin","password":"wrongpassword"}'"'"
echo ""
echo "Response:"
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"wrongpassword"}' | jq .
echo ""
echo "Expected: code=1002, message='invalid credentials', data=null, HTTP 401"
echo ""

# Test 4: /me without token
echo "========================================="
echo "Test 4: GET /api/v1/me (No Token)"
echo "========================================="
echo ""
echo "Command:"
echo "curl http://localhost:8080/api/v1/me"
echo ""
echo "Response:"
curl -s http://localhost:8080/api/v1/me | jq .
echo ""
echo "Expected: code=1001, message contains 'authorization', data=null, HTTP 401"
echo ""

# Test 5: /me with valid token
if [ -n "$TOKEN" ]; then
    echo "========================================="
    echo "Test 5: GET /api/v1/me (With Valid Token)"
    echo "========================================="
    echo ""
    echo "Command:"
    echo "curl -H \"Authorization: Bearer \$TOKEN\" http://localhost:8080/api/v1/me"
    echo ""
    echo "Response:"
    curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/me | jq .
    echo ""
    echo "Expected: code=0, message='success', data contains uid/username/role"
    echo ""
else
    echo "========================================="
    echo "Test 5: SKIPPED (No valid token)"
    echo "========================================="
    echo ""
fi

# Test 6: /me with invalid token
echo "========================================="
echo "Test 6: GET /api/v1/me (Invalid Token)"
echo "========================================="
echo ""
echo "Command:"
echo 'curl -H "Authorization: Bearer invalid.token.here" http://localhost:8080/api/v1/me'
echo ""
echo "Response:"
curl -s -H "Authorization: Bearer invalid.token.here" http://localhost:8080/api/v1/me | jq .
echo ""
echo "Expected: code=1002, message='invalid token', data=null, HTTP 401"
echo ""

echo "========================================="
echo "All tests completed!"
echo "========================================="
echo ""
echo "Verification checklist:"
echo "- Public routes (ping, login) work without auth"
echo "- Login returns JWT token"
echo "- Wrong password returns 401 + code=1002"
echo "- Protected routes require valid JWT"
echo "- Missing token returns 401 + code=1001"
echo "- Invalid token returns 401 + code=1002"
echo "- All responses use httpx unified format"
echo ""
