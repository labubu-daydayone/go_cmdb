-- Migration: Add enabled and status fields to node_ips table
-- Date: 2026-01-28

-- Add enabled field if not exists
SET @exist := (SELECT COUNT(*) FROM information_schema.COLUMNS 
               WHERE TABLE_SCHEMA = DATABASE() 
               AND TABLE_NAME = 'node_ips' 
               AND COLUMN_NAME = 'enabled');

SET @sql := IF(@exist = 0, 
    'ALTER TABLE node_ips ADD COLUMN enabled TINYINT(1) NOT NULL DEFAULT 1',
    'SELECT "Column enabled already exists" AS message');

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add status field if not exists
SET @exist := (SELECT COUNT(*) FROM information_schema.COLUMNS 
               WHERE TABLE_SCHEMA = DATABASE() 
               AND TABLE_NAME = 'node_ips' 
               AND COLUMN_NAME = 'status');

SET @sql := IF(@exist = 0, 
    'ALTER TABLE node_ips ADD COLUMN status ENUM(''active'',''unreachable'',''disabled'') NOT NULL DEFAULT ''active''',
    'SELECT "Column status already exists" AS message');

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
