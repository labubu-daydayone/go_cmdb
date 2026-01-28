-- Step 1: Create node_ips table
CREATE TABLE IF NOT EXISTS node_ips (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT NOT NULL,
    ip VARCHAR(64) NOT NULL,
    ip_type ENUM('main', 'sub') NOT NULL,
    enabled TINYINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_node_ips_ip (ip),
    UNIQUE KEY uk_node_ips_node_ip (node_id, ip),
    INDEX idx_node_ips_node_id (node_id),
    INDEX idx_node_ips_ip_type (ip_type),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Step 2: Migrate main IPs from nodes table to node_ips
INSERT INTO node_ips (node_id, ip, ip_type, enabled, created_at, updated_at)
SELECT id, main_ip, 'main', enabled, created_at, updated_at
FROM nodes
WHERE main_ip IS NOT NULL AND main_ip != '';

-- Step 3: Migrate sub IPs from node_sub_ips to node_ips
INSERT INTO node_ips (node_id, ip, ip_type, enabled, created_at, updated_at)
SELECT node_id, ip, 'sub', enabled, created_at, updated_at
FROM node_sub_ips;

-- Step 4: Create temporary mapping table for sub_ip_id to ip_id
CREATE TEMPORARY TABLE temp_subip_to_ipid AS
SELECT nsi.id AS old_sub_ip_id, ni.id AS new_ip_id
FROM node_sub_ips nsi
JOIN node_ips ni ON nsi.node_id = ni.node_id AND nsi.ip = ni.ip AND ni.ip_type = 'sub';

-- Step 5: Rename node_group_sub_ips to node_group_ips
RENAME TABLE node_group_sub_ips TO node_group_ips;

-- Step 6: Add new ip_id column to node_group_ips
ALTER TABLE node_group_ips ADD COLUMN ip_id BIGINT NULL AFTER node_group_id;

-- Step 7: Populate ip_id from sub_ip_id using temporary mapping
UPDATE node_group_ips ngi
JOIN temp_subip_to_ipid t ON ngi.sub_ip_id = t.old_sub_ip_id
SET ngi.ip_id = t.new_ip_id;

-- Step 8: Make ip_id NOT NULL and add constraints
ALTER TABLE node_group_ips 
    MODIFY COLUMN ip_id BIGINT NOT NULL,
    ADD UNIQUE KEY uk_node_group_ips (node_group_id, ip_id),
    ADD INDEX idx_node_group_ips_ip_id (ip_id),
    ADD FOREIGN KEY (ip_id) REFERENCES node_ips(id) ON DELETE CASCADE;

-- Step 9: Drop old sub_ip_id column
ALTER TABLE node_group_ips DROP COLUMN sub_ip_id;

-- Step 10: Drop old index on node_group_sub_ips (if exists)
-- Note: The table has been renamed, so we need to drop index on node_group_ips
ALTER TABLE node_group_ips DROP INDEX IF EXISTS idx_ng_subip;
