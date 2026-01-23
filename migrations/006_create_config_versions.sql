-- 006_create_config_versions.sql
-- Create config_versions table for configuration version management

CREATE TABLE IF NOT EXISTS config_versions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT 'Auto-increment ID',
    version BIGINT NOT NULL UNIQUE COMMENT 'Version number (equals to ID for global uniqueness)',
    node_id INT NOT NULL COMMENT 'Node ID',
    payload TEXT NOT NULL COMMENT 'Configuration payload (JSON)',
    reason VARCHAR(255) DEFAULT NULL COMMENT 'Reason for this version',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'Status: pending, applied, failed',
    last_error VARCHAR(255) DEFAULT NULL COMMENT 'Last error message if failed',
    applied_at DATETIME DEFAULT NULL COMMENT 'Applied timestamp',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created timestamp',
    
    INDEX idx_node_version (node_id, version),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Configuration versions table';
