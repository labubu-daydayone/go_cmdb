-- DNS Record Name Normalization Fix Script
-- Purpose: Fix existing FQDN names in domain_dns_records table to relative names
-- Author: Manus AI
-- Date: 2026-01-25

-- Step 1: Backup affected records before modification
-- Run this manually before executing the fix:
-- mysqldump -u admin -p cdn_control domain_dns_records --where="name LIKE '%.%'" > /tmp/dns_records_backup_$(date +%Y%m%d_%H%M%S).sql

-- Step 2: Preview records that will be modified
SELECT 
    r.id,
    r.domain_id,
    d.domain AS zone,
    r.name AS current_name,
    CASE
        -- If name equals zone, convert to @
        WHEN TRIM(TRAILING '.' FROM r.name) = TRIM(TRAILING '.' FROM d.domain) THEN '@'
        -- If name ends with .zone, extract relative part
        WHEN r.name LIKE CONCAT('%.', d.domain) THEN 
            TRIM(TRAILING '.' FROM SUBSTRING(r.name, 1, LENGTH(r.name) - LENGTH(d.domain) - 1))
        -- If name ends with zone (no dot), extract relative part
        WHEN r.name LIKE CONCAT('%', d.domain) AND r.name != d.domain THEN
            TRIM(TRAILING '.' FROM SUBSTRING(r.name, 1, LENGTH(r.name) - LENGTH(d.domain)))
        -- Otherwise, just remove trailing dot
        ELSE TRIM(TRAILING '.' FROM r.name)
    END AS new_name,
    r.type,
    r.value
FROM domain_dns_records r
INNER JOIN domains d ON r.domain_id = d.id
WHERE r.name LIKE '%.%'
   OR r.name LIKE '%.'
   OR TRIM(TRAILING '.' FROM r.name) = TRIM(TRAILING '.' FROM d.domain);

-- Step 3: Apply the fix (UPDATE statement)
-- WARNING: This will modify data. Ensure backup is complete before running!
UPDATE domain_dns_records r
INNER JOIN domains d ON r.domain_id = d.id
SET r.name = CASE
    -- If name equals zone, convert to @
    WHEN TRIM(TRAILING '.' FROM r.name) = TRIM(TRAILING '.' FROM d.domain) THEN '@'
    -- If name ends with .zone, extract relative part
    WHEN r.name LIKE CONCAT('%.', d.domain) THEN 
        TRIM(TRAILING '.' FROM SUBSTRING(r.name, 1, LENGTH(r.name) - LENGTH(d.domain) - 1))
    -- If name ends with zone (no dot), extract relative part
    WHEN r.name LIKE CONCAT('%', d.domain) AND r.name != d.domain THEN
        TRIM(TRAILING '.' FROM SUBSTRING(r.name, 1, LENGTH(r.name) - LENGTH(d.domain)))
    -- Otherwise, just remove trailing dot
    ELSE TRIM(TRAILING '.' FROM r.name)
END
WHERE r.name LIKE '%.%'
   OR r.name LIKE '%.'
   OR TRIM(TRAILING '.' FROM r.name) = TRIM(TRAILING '.' FROM d.domain);

-- Step 4: Verify the fix
SELECT 
    COUNT(*) AS total_records,
    SUM(CASE WHEN name LIKE '%.%' THEN 1 ELSE 0 END) AS records_with_dots,
    SUM(CASE WHEN name = '@' THEN 1 ELSE 0 END) AS root_records,
    SUM(CASE WHEN name NOT LIKE '%.%' AND name != '@' THEN 1 ELSE 0 END) AS relative_records
FROM domain_dns_records;

-- Step 5: Sample verification - show 10 random records
SELECT 
    r.id,
    d.domain AS zone,
    r.name,
    r.type,
    r.value,
    CONCAT(
        CASE 
            WHEN r.name = '@' THEN d.domain
            ELSE CONCAT(r.name, '.', d.domain)
        END
    ) AS fqdn
FROM domain_dns_records r
INNER JOIN domains d ON r.domain_id = d.id
ORDER BY RAND()
LIMIT 10;
