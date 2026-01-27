-- Add health check fields to nodes table

ALTER TABLE nodes
ADD COLUMN last_seen_at DATETIME NULL COMMENT 'Last successful health check time',
ADD COLUMN last_health_error VARCHAR(255) NULL COMMENT 'Last health check error message',
ADD COLUMN health_fail_count INT NOT NULL DEFAULT 0 COMMENT 'Consecutive health check failure count';

-- Add index for health check queries
CREATE INDEX idx_nodes_enabled_status ON nodes(enabled, status);
