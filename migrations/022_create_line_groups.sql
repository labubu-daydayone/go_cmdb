-- Migration: 022_create_line_groups.sql
-- Description: Create line_groups table for line group management
-- Date: 2026-01-28

CREATE TABLE IF NOT EXISTS `line_groups` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(128) NOT NULL COMMENT 'Line group name',
  `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Line group description',
  `domain_id` BIGINT NOT NULL COMMENT 'Foreign key to domains table',
  `node_group_id` BIGINT NOT NULL COMMENT 'Foreign key to node_groups table',
  `cname_prefix` VARCHAR(64) NOT NULL COMMENT 'CNAME prefix for this line group, must be unique',
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'Line group status: active, disabled',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT 'Creation timestamp',
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT 'Last update timestamp',
  UNIQUE KEY `uk_cname_prefix` (`cname_prefix`),
  KEY `idx_domain_id` (`domain_id`),
  KEY `idx_node_group_id` (`node_group_id`),
  CONSTRAINT `fk_line_groups_domain` FOREIGN KEY (`domain_id`) REFERENCES `domains` (`id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_line_groups_node_group` FOREIGN KEY (`node_group_id`) REFERENCES `node_groups` (`id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Line groups for CDN line management';
