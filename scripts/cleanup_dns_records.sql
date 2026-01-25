-- DNS Records Cleanup Script
-- Purpose: Clean up duplicate and FQDN records
-- Date: 2026-01-26

-- 1. Find and display duplicate records (same domain_id, type, name, value)
SELECT 
    domain_id, 
    type, 
    name, 
    value, 
    COUNT(*) as count,
    GROUP_CONCAT(id) as record_ids
FROM domain_dns_records
WHERE deleted_at IS NULL
GROUP BY domain_id, type, name, value
HAVING COUNT(*) > 1;

-- 2. Find records with FQDN in name field (should be relative names only)
SELECT 
    id,
    domain_id,
    type,
    name,
    value,
    status
FROM domain_dns_records
WHERE deleted_at IS NULL
  AND (
    name LIKE '%.com%' OR
    name LIKE '%.net%' OR
    name LIKE '%.org%' OR
    name LIKE '%.cn%' OR
    name LIKE '%.io%'
  );

-- 3. Delete duplicate records (keep the one with provider_record_id if exists)
-- Manual execution required: Review the duplicates first, then execute delete statements

-- 4. Find records with pending/error status for more than 7 days
SELECT 
    id,
    domain_id,
    type,
    name,
    value,
    status,
    desired_state,
    retry_count,
    last_error,
    created_at,
    updated_at
FROM domain_dns_records
WHERE deleted_at IS NULL
  AND status IN ('pending', 'error')
  AND updated_at < DATE_SUB(NOW(), INTERVAL 7 DAY);

-- 5. Find absent records that should have been deleted
SELECT 
    id,
    domain_id,
    type,
    name,
    value,
    status,
    desired_state,
    provider_record_id,
    created_at,
    updated_at
FROM domain_dns_records
WHERE deleted_at IS NULL
  AND desired_state = 'absent'
  AND updated_at < DATE_SUB(NOW(), INTERVAL 1 HOUR);
