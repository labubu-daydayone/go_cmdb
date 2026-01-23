-- Migration 009: Add certificate renew fields
-- T2-06: Certificate Auto-Renewal (Overwrite Update Mode)

-- Add renew-related fields to certificates table
ALTER TABLE certificates
ADD COLUMN issue_at DATETIME NULL COMMENT 'Certificate issue time',
ADD COLUMN expire_at DATETIME NOT NULL COMMENT 'Certificate expiration time',
ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'manual' COMMENT 'Certificate source: manual|acme',
ADD COLUMN renew_mode VARCHAR(20) NOT NULL DEFAULT 'manual' COMMENT 'Renewal mode: manual|auto',
ADD COLUMN acme_account_id INT NULL COMMENT 'ACME account ID for renewal',
ADD COLUMN renewing TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Renewal in progress flag',
ADD COLUMN last_error VARCHAR(500) NULL COMMENT 'Last error message',
ADD INDEX idx_acme_account_id (acme_account_id),
ADD INDEX idx_source_renew_mode (source, renew_mode),
ADD INDEX idx_expire_at (expire_at),
ADD INDEX idx_renewing (renewing);

-- Rename expires_at to expire_at for consistency (if exists)
-- Note: This assumes the old column name was expires_at
-- If the column is already named expire_at, this will fail gracefully
ALTER TABLE certificates
CHANGE COLUMN expires_at expire_at DATETIME NOT NULL COMMENT 'Certificate expiration time';

-- Add renew_cert_id field to certificate_requests table
ALTER TABLE certificate_requests
ADD COLUMN renew_cert_id INT NULL COMMENT 'Certificate ID for renewal (null for new certificate)',
ADD INDEX idx_renew_cert_id (renew_cert_id);

-- Add foreign key constraints (optional, for data integrity)
ALTER TABLE certificates
ADD CONSTRAINT fk_certificates_acme_account
FOREIGN KEY (acme_account_id) REFERENCES acme_accounts(id)
ON DELETE SET NULL;

ALTER TABLE certificate_requests
ADD CONSTRAINT fk_certificate_requests_renew_cert
FOREIGN KEY (renew_cert_id) REFERENCES certificates(id)
ON DELETE CASCADE;
