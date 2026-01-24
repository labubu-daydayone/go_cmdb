-- Migration: 014_create_domain_dns_zone_meta
-- Purpose: Create domain_dns_zone_meta table for DNS zone metadata and NS cache

CREATE TABLE domain_dns_zone_meta (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  domain_id BIGINT NOT NULL UNIQUE,
  name_servers_json JSON NOT NULL,
  last_sync_at DATETIME NOT NULL,
  last_error VARCHAR(255) NULL,
  created_at DATETIME(3),
  updated_at DATETIME(3),
  KEY idx_domain_id (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
