-- T2-01 Agent Tasks SQL Verification Script

-- 1. Verify agent_tasks table structure
DESC agent_tasks;

-- 2. Verify pending->running->success status flow
SELECT id, node_id, type, status, attempts, created_at, updated_at
FROM agent_tasks
WHERE type = 'apply_config'
ORDER BY id DESC
LIMIT 5;

-- 3. Verify failed tasks with last_error
SELECT id, node_id, type, status, last_error, attempts
FROM agent_tasks
WHERE status = 'failed'
ORDER BY id DESC
LIMIT 5;

-- 4. Verify attempts and next_retry_at for retried tasks
SELECT id, type, status, attempts, next_retry_at, last_error
FROM agent_tasks
WHERE attempts > 1
ORDER BY id DESC
LIMIT 5;

-- 5. Verify request_id unique constraint
SELECT request_id, COUNT(*) as count
FROM agent_tasks
GROUP BY request_id
HAVING count > 1;

-- 6. Verify task distribution by type
SELECT type, COUNT(*) as count, 
       SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM agent_tasks
GROUP BY type;

-- 7. Verify task distribution by node
SELECT node_id, COUNT(*) as total_tasks,
       SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM agent_tasks
GROUP BY node_id
ORDER BY total_tasks DESC;

-- 8. Verify recent task execution timeline
SELECT id, node_id, type, status, attempts, 
       TIMESTAMPDIFF(SECOND, created_at, updated_at) as execution_time_seconds,
       created_at, updated_at
FROM agent_tasks
ORDER BY id DESC
LIMIT 10;
