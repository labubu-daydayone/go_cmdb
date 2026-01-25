-- DNS Records Database Constraints
-- Purpose: Add unique constraint on provider_record_id
-- Date: 2026-01-26

-- 1. Add unique index on provider_record_id (if not exists)
-- This ensures one record_id can only exist once in the database
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_record_id 
ON domain_dns_records(provider_record_id)
WHERE provider_record_id IS NOT NULL AND provider_record_id != '';

-- 2. Add composite index for faster lookups
-- Used by Pull Sync to find records by (domain_id, type, name, value)
CREATE INDEX IF NOT EXISTS idx_domain_type_name_value 
ON domain_dns_records(domain_id, type, name, value);

-- 3. Add index on desired_state for faster deletion queries
CREATE INDEX IF NOT EXISTS idx_desired_state 
ON domain_dns_records(desired_state);

-- 4. Add index on status for faster pending/error queries
CREATE INDEX IF NOT EXISTS idx_status 
ON domain_dns_records(status);

-- Note: MySQL does not support partial indexes (WHERE clause)
-- If using PostgreSQL, the WHERE clause in step 1 would work
-- For MySQL, we need to handle NULL values in application logic
