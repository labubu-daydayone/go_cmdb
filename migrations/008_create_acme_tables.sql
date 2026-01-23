-- 008_create_acme_tables.sql
-- Create ACME-related tables for certificate automation

-- ACME Providers (Let's Encrypt, Google Public CA, etc.)
CREATE TABLE IF NOT EXISTS acme_providers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE COMMENT 'Provider name (letsencrypt, google)',
    directory_url VARCHAR(255) NOT NULL COMMENT 'ACME directory URL',
    requires_eab BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Requires External Account Binding',
    status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT 'active|inactive',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='ACME providers';

-- ACME Accounts
CREATE TABLE IF NOT EXISTS acme_accounts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    provider_id INT NOT NULL COMMENT 'Reference to acme_providers.id',
    email VARCHAR(255) NOT NULL COMMENT 'Account email',
    account_key_pem TEXT NOT NULL COMMENT 'Private key for ACME account',
    registration_uri VARCHAR(500) COMMENT 'ACME registration URI',
    eab_kid VARCHAR(255) COMMENT 'External Account Binding Key ID',
    eab_hmac_key TEXT COMMENT 'External Account Binding HMAC Key (encrypted)',
    eab_expires_at TIMESTAMP NULL COMMENT 'EAB expiration time',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'pending|active|inactive',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_provider_email (provider_id, email),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='ACME accounts';

-- Certificate Requests (ACME orders)
CREATE TABLE IF NOT EXISTS certificate_requests (
    id INT AUTO_INCREMENT PRIMARY KEY,
    account_id INT NOT NULL COMMENT 'Reference to acme_accounts.id',
    domains TEXT NOT NULL COMMENT 'JSON array of domains',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|success|failed',
    attempts INT NOT NULL DEFAULT 0 COMMENT 'Retry attempts',
    poll_max_attempts INT NOT NULL DEFAULT 10 COMMENT 'Max retry attempts',
    last_error TEXT COMMENT 'Last error message',
    result_certificate_id INT COMMENT 'Reference to certificates.id',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_account (account_id),
    INDEX idx_status (status),
    INDEX idx_result_cert (result_certificate_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Certificate requests';

-- Certificate Domains (SAN)
CREATE TABLE IF NOT EXISTS certificate_domains (
    id INT AUTO_INCREMENT PRIMARY KEY,
    certificate_id INT NOT NULL COMMENT 'Reference to certificates.id',
    domain VARCHAR(255) NOT NULL COMMENT 'Domain name (e.g., example.com or *.example.com)',
    is_wildcard BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Is wildcard domain',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_cert_domain (certificate_id, domain),
    INDEX idx_domain (domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Certificate domains (SAN)';

-- Certificate Bindings (website <-> certificate)
CREATE TABLE IF NOT EXISTS certificate_bindings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    certificate_id INT NOT NULL COMMENT 'Reference to certificates.id',
    website_id INT NOT NULL COMMENT 'Reference to websites.id',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive' COMMENT 'inactive|active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_cert_website (certificate_id, website_id),
    UNIQUE INDEX idx_website_unique (website_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Certificate bindings';

-- Update certificates table (add new fields)
ALTER TABLE certificates 
    ADD COLUMN IF NOT EXISTS fingerprint VARCHAR(64) COMMENT 'SHA256 fingerprint',
    ADD COLUMN IF NOT EXISTS chain_pem TEXT COMMENT 'Certificate chain',
    ADD COLUMN IF NOT EXISTS issuer VARCHAR(255) COMMENT 'Certificate issuer';

-- Add unique index on fingerprint
ALTER TABLE certificates ADD UNIQUE INDEX idx_fingerprint (fingerprint);

-- Insert default ACME providers
INSERT INTO acme_providers (name, directory_url, requires_eab, status) VALUES
    ('letsencrypt', 'https://acme-v02.api.letsencrypt.org/directory', FALSE, 'active'),
    ('google', 'https://dv.acme-v02.api.pki.goog/directory', TRUE, 'active')
ON DUPLICATE KEY UPDATE 
    directory_url = VALUES(directory_url),
    requires_eab = VALUES(requires_eab);
