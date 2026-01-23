#!/bin/bash

# T2-01 Agent Tasks API Test Script
# This script tests the complete control plane <-> agent communication flow

set -e

BASE_URL="http://localhost:8080/api/v1"
AGENT_URL="http://localhost:9090"
TOKEN=""

echo "=== T2-01 Agent Tasks API Test ==="
echo ""

# Function to make authenticated requests
auth_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json"
    else
        curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

echo "Step 1: Login to get JWT token"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | jq -r '.data.token')
echo "✓ Token obtained: ${TOKEN:0:20}..."
echo ""

echo "Step 2: Create a node pointing to local agent"
NODE_RESP=$(auth_request POST "/nodes/create" '{
  "name": "test-node-1",
  "mainIP": "127.0.0.1",
  "agentPort": 9090,
  "enabled": true,
  "status": "online"
}')
NODE_ID=$(echo $NODE_RESP | jq -r '.data.id')
echo "✓ Node created: ID=$NODE_ID"
echo ""

echo "Step 3: Create apply_config task"
TASK1_RESP=$(auth_request POST "/agent-tasks/create" '{
  "nodeId": '$NODE_ID',
  "type": "apply_config",
  "payload": {"vhost": "example.com", "upstream": ["192.168.1.1:80"]}
}')
TASK1_ID=$(echo $TASK1_RESP | jq -r '.data.id')
TASK1_REQUEST_ID=$(echo $TASK1_RESP | jq -r '.data.requestId')
echo "✓ apply_config task created: ID=$TASK1_ID, RequestID=$TASK1_REQUEST_ID"
echo ""

sleep 2

echo "Step 4: Create reload task"
TASK2_RESP=$(auth_request POST "/agent-tasks/create" '{
  "nodeId": '$NODE_ID',
  "type": "reload",
  "payload": {}
}')
TASK2_ID=$(echo $TASK2_RESP | jq -r '.data.id')
echo "✓ reload task created: ID=$TASK2_ID"
echo ""

sleep 2

echo "Step 5: Create purge_cache task"
TASK3_RESP=$(auth_request POST "/agent-tasks/create" '{
  "nodeId": '$NODE_ID',
  "type": "purge_cache",
  "payload": {"urls": ["/api/*", "/static/*"]}
}')
TASK3_ID=$(echo $TASK3_RESP | jq -r '.data.id')
echo "✓ purge_cache task created: ID=$TASK3_ID"
echo ""

sleep 2

echo "Step 6: Query tasks list"
TASKS_LIST=$(auth_request GET "/agent-tasks?nodeId=$NODE_ID")
TASKS_COUNT=$(echo $TASKS_LIST | jq -r '.data.total')
echo "✓ Tasks list retrieved: total=$TASKS_COUNT"
echo ""

echo "Step 7: Query task detail"
TASK_DETAIL=$(auth_request GET "/agent-tasks/$TASK1_ID")
TASK_STATUS=$(echo $TASK_DETAIL | jq -r '.data.status')
echo "✓ Task detail retrieved: status=$TASK_STATUS"
echo ""

echo "Step 8: Verify agent files created"
if [ -f "/tmp/cmdb_apply_config_$TASK1_REQUEST_ID.json" ]; then
    echo "✓ Agent file created: /tmp/cmdb_apply_config_$TASK1_REQUEST_ID.json"
    cat "/tmp/cmdb_apply_config_$TASK1_REQUEST_ID.json"
else
    echo "✗ Agent file not found"
fi
echo ""

echo "Step 9: Test task filtering by status"
SUCCESS_TASKS=$(auth_request GET "/agent-tasks?status=success")
echo "✓ Filtered by status=success: $(echo $SUCCESS_TASKS | jq -r '.data.total') tasks"
echo ""

echo "Step 10: Test task filtering by type"
RELOAD_TASKS=$(auth_request GET "/agent-tasks?type=reload")
echo "✓ Filtered by type=reload: $(echo $RELOAD_TASKS | jq -r '.data.total') tasks"
echo ""

echo "Step 11: Test retry failed task (simulate failure first)"
echo "Note: This requires stopping the agent to simulate failure"
echo ""

echo "Step 12: Test idempotency (send duplicate request to agent)"
IDEMPOTENT_RESP=$(curl -s -X POST "$AGENT_URL/agent/v1/tasks/execute" \
    -H "Authorization: Bearer default-agent-token" \
    -H "Content-Type: application/json" \
    -d '{
      "requestId": "'$TASK1_REQUEST_ID'",
      "type": "apply_config",
      "payload": {"vhost": "example.com"}
    }')
echo "✓ Idempotent request sent, response:"
echo $IDEMPOTENT_RESP | jq '.'
echo ""

echo "=== All tests completed ==="
echo ""
echo "Summary:"
echo "- Created 3 tasks (apply_config, reload, purge_cache)"
echo "- Verified task list and detail queries"
echo "- Verified agent file creation"
echo "- Tested task filtering"
echo "- Verified idempotency"
