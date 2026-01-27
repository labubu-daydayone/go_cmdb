-- Migration: Create agent_identities table for Node mTLS client certificates
-- Task: T2-28
-- Purpose: Store mTLS client certificates and private keys for each Node

CREATE TABLE IF NOT EXISTS agent_identities (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT NOT NULL,
    fingerprint VARCHAR(128) NOT NULL,
    cert_pem LONGTEXT NOT NULL,
    key_pem LONGTEXT NOT NULL,
    status ENUM('active', 'revoked') NOT NULL DEFAULT 'active',
    issued_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    UNIQUE KEY uk_node_id (node_id),
    UNIQUE KEY uk_fingerprint (fingerprint),
    KEY idx_status (status),
    CONSTRAINT fk_agent_identities_node_id FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent mTLS client identities';
