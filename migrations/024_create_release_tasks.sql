-- Migration: Create release_tasks table
-- Purpose: Fix executor startup error "Table 'cdn_control.release_tasks' doesn't exist"
-- Date: 2026-01-29

CREATE TABLE IF NOT EXISTS release_tasks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    type VARCHAR(64) NOT NULL DEFAULT 'apply_config',
    status VARCHAR(32) NOT NULL,
    payload JSON NULL,
    last_error TEXT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    next_retry_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_release_tasks_status_id (status, id),
    KEY idx_release_tasks_next_retry_at (next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Rollback SQL:
-- DROP TABLE IF NOT EXISTS release_tasks;
