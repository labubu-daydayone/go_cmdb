-- 007_create_domain_dns_records.sql
-- Create domain_dns_records table for DNS Worker

CREATE TABLE IF NOT EXISTS domain_dns_records (
    id INT AUTO_INCREMENT PRIMARY KEY,
    domain_id INT NOT NULL,
    type ENUM('A', 'AAAA', 'CNAME', 'TXT') NOT NULL,
    name VARCHAR(255) NOT NULL COMMENT 'Relative name: @ / www / a.b',
    value VARCHAR(2048) NOT NULL,
    ttl INT DEFAULT 120,
    proxied TINYINT DEFAULT 0 COMMENT 'Cloudflare proxy (orange cloud), 0=DNS only',
    status ENUM('pending', 'active', 'error', 'running') DEFAULT 'pending',
    desired_state ENUM('present', 'absent') NOT NULL DEFAULT 'present' COMMENT 'present=should exist, absent=should be deleted',
    provider_record_id VARCHAR(128) NULL COMMENT 'Cloudflare record ID',
    last_error VARCHAR(255) NULL,
    retry_count INT DEFAULT 0,
    next_retry_at DATETIME NULL,
    owner_type ENUM('node_group', 'line_group', 'website_domain', 'acme_challenge') NOT NULL,
    owner_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_domain_id (domain_id),
    INDEX idx_domain_type_name (domain_id, type, name),
    INDEX idx_owner (owner_type, owner_id),
    INDEX idx_status_desired (status, desired_state),
    INDEX idx_next_retry (next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='DNS records for Cloudflare sync';

-- Business-level unique constraint (enforced in application layer):
-- unique(domain_id, type, name, value, owner_type, owner_id, desired_state)
