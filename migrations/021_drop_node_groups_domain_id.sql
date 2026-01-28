-- Migration: 021_drop_node_groups_domain_id.sql
-- Purpose: Remove domain_id from node_groups table
-- Reason: Node Group should only describe IP groups, domain relationship should only exist in domain_dns_records

-- Drop foreign key constraint first
ALTER TABLE `node_groups` DROP FOREIGN KEY `fk_node_groups_domain`;

-- Drop index on domain_id
ALTER TABLE `node_groups` DROP INDEX `idx_node_groups_domain_id`;

-- Drop domain_id column
ALTER TABLE `node_groups` DROP COLUMN `domain_id`;

-- Drop Domain relation from model (this is a code-level change, not SQL)
-- The Domain field in NodeGroup struct will be removed in code
