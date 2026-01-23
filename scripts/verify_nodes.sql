-- SQL verification script for T1-01 nodes management
-- Execute these commands to verify nodes and sub IPs functionality

-- 1. Check if nodes table exists
SHOW TABLES LIKE 'nodes';

-- 2. Check if node_sub_ips table exists
SHOW TABLES LIKE 'node_sub_ips';

-- 3. Describe nodes table structure
DESC nodes;

-- 4. Describe node_sub_ips table structure
DESC node_sub_ips;

-- 5. Check indexes on nodes table
SHOW INDEX FROM nodes;

-- 6. Check indexes on node_sub_ips table
SHOW INDEX FROM node_sub_ips;

-- 7. Count total nodes
SELECT COUNT(*) AS total_nodes FROM nodes;

-- 8. Count total sub IPs
SELECT COUNT(*) AS total_sub_ips FROM node_sub_ips;

-- 9. List all nodes with their sub IP count
SELECT 
    n.id,
    n.name,
    n.main_ip,
    n.agent_port,
    n.enabled,
    n.status,
    COUNT(s.id) AS sub_ip_count
FROM nodes n
LEFT JOIN node_sub_ips s ON n.id = s.node_id
GROUP BY n.id
ORDER BY n.id DESC;

-- 10. List all sub IPs grouped by node
SELECT 
    n.id AS node_id,
    n.name AS node_name,
    s.id AS sub_ip_id,
    s.ip AS sub_ip,
    s.enabled AS sub_ip_enabled
FROM nodes n
LEFT JOIN node_sub_ips s ON n.id = s.node_id
ORDER BY n.id, s.id;

-- 11. Test IP search (main IP)
SELECT 
    n.id,
    n.name,
    n.main_ip,
    'main_ip' AS match_type
FROM nodes n
WHERE n.main_ip LIKE '%192.168.1%';

-- 12. Test IP search (sub IP)
SELECT 
    n.id,
    n.name,
    n.main_ip,
    s.ip AS sub_ip,
    'sub_ip' AS match_type
FROM nodes n
INNER JOIN node_sub_ips s ON n.id = s.node_id
WHERE s.ip LIKE '%192.168.1%';

-- 13. Verify cascade delete (check foreign key constraint)
SELECT 
    CONSTRAINT_NAME,
    TABLE_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME,
    DELETE_RULE
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'node_sub_ips'
  AND REFERENCED_TABLE_NAME = 'nodes';

-- 14. Check enabled/disabled distribution
SELECT 
    enabled,
    COUNT(*) AS count
FROM nodes
GROUP BY enabled;

-- 15. Check status distribution
SELECT 
    status,
    COUNT(*) AS count
FROM nodes
GROUP BY status;
