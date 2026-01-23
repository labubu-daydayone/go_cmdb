-- SQL verification script for T0-04 migration
-- Execute these commands to verify migration success

-- 1. Check if users table exists
SHOW TABLES LIKE 'users';

-- 2. Check if domain_dns_records table exists
SHOW TABLES LIKE 'domain_dns_records';

-- 3. Describe domain_dns_records table structure
DESC domain_dns_records;

-- 4. List all tables created by migration
SHOW TABLES;

-- 5. Check indexes on users table
SHOW INDEX FROM users;

-- 6. Check indexes on domain_dns_records table
SHOW INDEX FROM domain_dns_records;

-- 7. Check indexes on domain_dns_providers table
SHOW INDEX FROM domain_dns_providers;

-- 8. Verify unique constraint on domains.domain
SHOW INDEX FROM domains WHERE Key_name = 'domain';

-- 9. Count records in each table (should be 0 after fresh migration)
SELECT 'users' AS table_name, COUNT(*) AS record_count FROM users
UNION ALL
SELECT 'api_keys', COUNT(*) FROM api_keys
UNION ALL
SELECT 'domains', COUNT(*) FROM domains
UNION ALL
SELECT 'domain_dns_providers', COUNT(*) FROM domain_dns_providers
UNION ALL
SELECT 'domain_dns_records', COUNT(*) FROM domain_dns_records;
