-- Migration: Create acme_provider_defaults table
-- Purpose: Store default ACME account for each provider
-- Date: 2026-01-26

CREATE TABLE IF NOT EXISTS acme_provider_defaults (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    provider_id BIGINT NOT NULL UNIQUE COMMENT 'ACME provider ID (FK to acme_providers.id)',
    account_id BIGINT NOT NULL COMMENT 'Default ACME account ID (FK to acme_accounts.id)',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    
    -- Foreign keys
    CONSTRAINT fk_provider_defaults_provider FOREIGN KEY (provider_id) REFERENCES acme_providers(id) ON DELETE CASCADE,
    CONSTRAINT fk_provider_defaults_account FOREIGN KEY (account_id) REFERENCES acme_accounts(id) ON DELETE CASCADE,
    
    -- Indexes
    INDEX idx_account_id (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='ACME provider default accounts';

-- Rollback SQL:
-- DROP TABLE IF EXISTS acme_provider_defaults;
