#!/bin/bash

# Test script for go_cmdb
# This script demonstrates the testing process

set -e

echo "========================================="
echo "go_cmdb Test Suite"
echo "========================================="
echo ""

# 1. Run unit tests
echo "1. Running unit tests..."
go test ./... -v
echo "✓ Unit tests passed"
echo ""

# 2. Build the application
echo "2. Building application..."
go build -o bin/cmdb ./cmd/cmdb
echo "✓ Build successful"
echo ""

# 3. Verify binary
echo "3. Verifying binary..."
ls -lh bin/cmdb
file bin/cmdb
echo "✓ Binary verified"
echo ""

echo "========================================="
echo "Manual Test Instructions"
echo "========================================="
echo ""
echo "To test the application manually:"
echo ""
echo "1. Ensure MySQL and Redis are running"
echo ""
echo "2. Configure environment variables:"
echo "   export MYSQL_DSN='user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local'"
echo "   export REDIS_ADDR='localhost:6379'"
echo "   export REDIS_PASS=''"
echo "   export REDIS_DB='0'"
echo "   export HTTP_ADDR=':8080'"
echo ""
echo "3. Start the server:"
echo "   ./bin/cmdb"
echo ""
echo "4. Test the ping endpoint:"
echo "   curl http://localhost:8080/api/v1/ping"
echo ""
echo "Expected response:"
echo '   {"code":0,"message":"pong"}'
echo ""
echo "========================================="
echo "Rollback Strategy"
echo "========================================="
echo ""
echo "The application follows a fail-fast strategy:"
echo "- MySQL connection failure → exits with code 1"
echo "- Redis connection failure → exits with code 1"
echo "- Config loading failure → exits with code 1"
echo ""
echo "This ensures the application never runs in a"
echo "partially initialized state."
echo ""
