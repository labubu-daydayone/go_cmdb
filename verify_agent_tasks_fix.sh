#!/bin/bash

# --- Helper Functions ---
API_BASE_URL="http://20.2.140.226:8080/api/v1/agent/tasks"
NODE_ID=123

log_step() {
    echo -e "\n--- $1 ---"
}

run_mysql() {
    mysql -h 20.2.140.226 -u admin -pdeag2daicahThiipheed4gi4 -D cdn_control -N -e "$1"
}

# --- Test Execution ---

log_step "1. Initial Cleanup"
run_mysql "DELETE FROM agent_tasks WHERE node_id = ${NODE_ID};"
run_mysql "DELETE FROM nodes WHERE id = ${NODE_ID};"
run_mysql "INSERT INTO nodes (id, name, main_ip, agent_port, enabled, status, created_at, updated_at) VALUES (${NODE_ID}, 'test-node-123', '127.0.0.1', 8080, 1, 'online', NOW(), NOW());"
echo "Cleanup complete."

# --- Test Case 1: Full Success Lifecycle (pending -> running -> succeeded) ---
log_step "2. Creating a new 'pending' task in DB"
run_mysql "INSERT INTO agent_tasks (node_id, type, payload, status, created_at, updated_at) VALUES (${NODE_ID}, 'deploy', '{\"version\": \"1.0\"}', 'pending', NOW(), NOW());"
TASK_ID=$(run_mysql "SELECT id FROM agent_tasks WHERE node_id = ${NODE_ID} AND status = 'pending' ORDER BY id DESC LIMIT 1;")
echo "Created task with ID: ${TASK_ID}"

log_step "3. Agent pulls the task (pending -> running)"
PULL_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/pull" -H "X-Node-Id: ${NODE_ID}")
echo "API Response from /pull:"
echo ${PULL_RESPONSE} | jq

# Verification
PULLED_STATUS=$(echo ${PULL_RESPONSE} | jq -r '.data.items[0].status')
DB_STATUS_AFTER_PULL=$(run_mysql "SELECT status FROM agent_tasks WHERE id = ${TASK_ID};")
echo "- API returned status: ${PULLED_STATUS}"
echo "- DB status after pull: ${DB_STATUS_AFTER_PULL}"
if [[ "${PULLED_STATUS}" == "running" && "${DB_STATUS_AFTER_PULL}" == "running" ]]; then
    echo "✅ PASSED: Task correctly transitioned to 'running'."
else
    echo "❌ FAILED: Task did not transition to 'running' correctly."
    exit 1
fi

log_step "4. Agent reports task as 'succeeded' (running -> success)"
UPDATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/update-status" -H "Content-Type: application/json" -H "X-Node-Id: ${NODE_ID}" -d "{\"taskId\": ${TASK_ID}, \"status\": \"succeeded\"}")
echo "API Response from /update-status:"
echo ${UPDATE_RESPONSE} | jq

# Verification
DB_STATUS_AFTER_UPDATE=$(run_mysql "SELECT status FROM agent_tasks WHERE id = ${TASK_ID};")
echo "- DB status after update: ${DB_STATUS_AFTER_UPDATE}"
if [[ "${DB_STATUS_AFTER_UPDATE}" == "success" ]]; then
    echo "✅ PASSED: Task correctly updated to 'success' in DB."
else
    echo "❌ FAILED: DB status should be 'success', but is '${DB_STATUS_AFTER_UPDATE}'."
    exit 1
fi

# --- Test Case 2: Status Mapping Verification (DB 'success' -> API 'succeeded') ---
log_step "5. Agent pulls again to verify 'success' -> 'succeeded' mapping"
# The service was modified to also pull 'success' tasks for this verification
PULL_SUCCESS_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/pull" -H "X-Node-Id: ${NODE_ID}")
echo "API Response from /pull (expecting succeeded task):"
echo ${PULL_SUCCESS_RESPONSE} | jq

# Verification
MAPPED_STATUS=$(echo ${PULL_SUCCESS_RESPONSE} | jq -r '.data.items[] | select(.id=='${TASK_ID}') | .status')
echo "- API returned status for completed task: ${MAPPED_STATUS}"
if [[ "${MAPPED_STATUS}" == "succeeded" ]]; then
    echo "✅ PASSED: DB 'success' was correctly mapped to API 'succeeded'."
else
    echo "❌ FAILED: Status mapping is incorrect. Expected 'succeeded', got '${MAPPED_STATUS}'."
    exit 1
fi

# --- Test Case 3: Failure Lifecycle (pending -> running -> failed) ---
log_step "6. Creating a second 'pending' task for failure test"
run_mysql "INSERT INTO agent_tasks (node_id, type, payload, status, created_at, updated_at) VALUES (${NODE_ID}, 'cleanup', '{\"target\": \"/tmp\"}', 'pending', NOW(), NOW());"
FAIL_TASK_ID=$(run_mysql "SELECT id FROM agent_tasks WHERE node_id = ${NODE_ID} AND status = 'pending' ORDER BY id DESC LIMIT 1;")
echo "Created task with ID: ${FAIL_TASK_ID}"

log_step "7. Agent pulls the second task"
curl -s -X GET "${API_BASE_URL}/pull" -H "X-Node-Id: ${NODE_ID}" > /dev/null

log_step "8. Agent reports task as 'failed'"
ERROR_MSG="Disk is full"
curl -s -X POST "${API_BASE_URL}/update-status" -H "Content-Type: application/json" -H "X-Node-Id: ${NODE_ID}" -d "{\"taskId\": ${FAIL_TASK_ID}, \"status\": \"failed\", \"errorMessage\": \"${ERROR_MSG}\"}" > /dev/null

# Verification
DB_STATUS_AFTER_FAIL=$(run_mysql "SELECT status FROM agent_tasks WHERE id = ${FAIL_TASK_ID};")
DB_ERROR_MSG=$(run_mysql "SELECT last_error FROM agent_tasks WHERE id = ${FAIL_TASK_ID};")
echo "- DB status after fail: ${DB_STATUS_AFTER_FAIL}"
echo "- DB error message: ${DB_ERROR_MSG}"
if [[ "${DB_STATUS_AFTER_FAIL}" == "failed" && "${DB_ERROR_MSG}" == "${ERROR_MSG}" ]]; then
    echo "✅ PASSED: Task correctly updated to 'failed' with error message."
else
    echo "❌ FAILED: Task failure update was not correct."
    exit 1
fi

# --- Test Case 4: Auth Failure ---
log_step "9. Testing request without X-Node-Id header"
AUTH_FAIL_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/pull")
AUTH_FAIL_CODE=$(echo ${AUTH_FAIL_RESPONSE} | jq -r '.code')
echo "API Response:"
echo ${AUTH_FAIL_RESPONSE} | jq
if [[ "${AUTH_FAIL_CODE}" == "2001" ]]; then
    echo "✅ PASSED: API correctly rejected request without auth header."
else
    echo "❌ FAILED: API should have returned code 2001, but got ${AUTH_FAIL_CODE}."
    exit 1
fi

log_step "All tests passed successfully!"
