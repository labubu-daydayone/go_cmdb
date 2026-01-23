#!/bin/bash

# Test script for T0-02 httpx implementation
# This script tests the unified response structure and error codes

set -e

echo "========================================="
echo "T0-02 httpx Test Suite"
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

# Test 1: Ping success (code=0)
echo "========================================="
echo "Test 1: GET /api/v1/ping (Success)"
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

# Test 2: Demo error (code=5001, HTTP 500)
echo "========================================="
echo "Test 2: GET /api/v1/demo/error (Internal Error)"
echo "========================================="
echo ""
echo "Command:"
echo "curl http://localhost:8080/api/v1/demo/error"
echo ""
echo "Response:"
curl -s http://localhost:8080/api/v1/demo/error | jq .
echo ""
echo "Expected: code=5001, message='internal error', data=null, HTTP 500"
echo ""

# Test 3: Parameter missing (code=2001, HTTP 400)
echo "========================================="
echo "Test 3: GET /api/v1/demo/param (Parameter Missing)"
echo "========================================="
echo ""
echo "Command:"
echo "curl 'http://localhost:8080/api/v1/demo/param'"
echo ""
echo "Response:"
curl -s 'http://localhost:8080/api/v1/demo/param' | jq .
echo ""
echo "Expected: code=2001, message contains 'required', data=null, HTTP 400"
echo ""

# Test 4: Parameter success
echo "========================================="
echo "Test 4: GET /api/v1/demo/param?x=test (Success)"
echo "========================================="
echo ""
echo "Command:"
echo "curl 'http://localhost:8080/api/v1/demo/param?x=test'"
echo ""
echo "Response:"
curl -s 'http://localhost:8080/api/v1/demo/param?x=test' | jq .
echo ""
echo "Expected: code=0, message='success', data.x='test'"
echo ""

# Test 5: Not found (code=3001, HTTP 404)
echo "========================================="
echo "Test 5: GET /api/v1/demo/notfound (Not Found)"
echo "========================================="
echo ""
echo "Command:"
echo "curl http://localhost:8080/api/v1/demo/notfound"
echo ""
echo "Response:"
curl -s http://localhost:8080/api/v1/demo/notfound | jq .
echo ""
echo "Expected: code=3001, message='resource not found', data=null, HTTP 404"
echo ""

echo "========================================="
echo "All tests completed!"
echo "========================================="
echo ""
echo "Verification checklist:"
echo "- All responses follow unified format (code, message, data)"
echo "- Success responses have code=0, HTTP 200"
echo "- Error responses have code!=0, data=null"
echo "- HTTP status codes match error semantics"
echo "- No handler directly uses c.JSON to construct responses"
echo ""
