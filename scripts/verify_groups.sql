-- T1-02 SQL Verification Script
-- This script verifies the database state for node groups and line groups

-- 1. Check node_groups table exists
SHOW TABLES LIKE 'node_groups';

-- 2. Check line_groups table exists
SHOW TABLES LIKE 'line_groups';

-- 3. Check node_group_sub_ips table exists
SHOW TABLES LIKE 'node_group_sub_ips';

-- 4. Describe node_groups table structure
DESC node_groups;

-- 5. Describe line_groups table structure
DESC line_groups;

-- 6. Describe node_group_sub_ips table structure
DESC node_group_sub_ips;

-- 7. Show indexes on node_groups
SHOW INDEX FROM node_groups;

-- 8. Show indexes on line_groups
SHOW INDEX FROM line_groups;

-- 9. Show indexes on node_group_sub_ips
SHOW INDEX FROM node_group_sub_ips;

-- 10. List all node groups with sub IP count
SELECT 
    ng.id,
    ng.name,
    ng.cname_prefix,
    ng.cname,
    ng.status,
    COUNT(ngs.id) AS sub_ip_count
FROM node_groups ng
LEFT JOIN node_group_sub_ips ngs ON ng.id = ngs.node_group_id
GROUP BY ng.id
ORDER BY ng.id DESC
LIMIT 10;

-- 11. List all line groups with node group info
SELECT 
    lg.id,
    lg.name,
    lg.cname_prefix,
    lg.cname,
    lg.status,
    ng.name AS node_group_name,
    ng.cname AS node_group_cname
FROM line_groups lg
LEFT JOIN node_groups ng ON lg.node_group_id = ng.id
ORDER BY lg.id DESC
LIMIT 10;

-- 12. Verify CNAME format (cname = cname_prefix + "." + domain.domain)
SELECT 
    ng.id,
    ng.name,
    ng.cname_prefix,
    ng.cname,
    d.domain,
    CONCAT(ng.cname_prefix, '.', d.domain) AS expected_cname,
    CASE 
        WHEN ng.cname = CONCAT(ng.cname_prefix, '.', d.domain) THEN 'OK'
        ELSE 'MISMATCH'
    END AS validation
FROM node_groups ng
JOIN domains d ON ng.domain_id = d.id
LIMIT 10;

-- 13. Verify line group CNAME format
SELECT 
    lg.id,
    lg.name,
    lg.cname_prefix,
    lg.cname,
    d.domain,
    CONCAT(lg.cname_prefix, '.', d.domain) AS expected_cname,
    CASE 
        WHEN lg.cname = CONCAT(lg.cname_prefix, '.', d.domain) THEN 'OK'
        ELSE 'MISMATCH'
    END AS validation
FROM line_groups lg
JOIN domains d ON lg.domain_id = d.id
LIMIT 10;

-- 14. List DNS A records for node groups
SELECT 
    dr.id,
    dr.type,
    dr.name,
    dr.value,
    dr.owner_type,
    dr.owner_id,
    dr.status,
    dr.last_error,
    ng.name AS node_group_name
FROM domain_dns_records dr
JOIN node_groups ng ON dr.owner_id = ng.id
WHERE dr.owner_type = 'node_group'
ORDER BY dr.id DESC
LIMIT 20;

-- 15. List DNS CNAME records for line groups
SELECT 
    dr.id,
    dr.type,
    dr.name,
    dr.value,
    dr.owner_type,
    dr.owner_id,
    dr.status,
    dr.last_error,
    lg.name AS line_group_name
FROM domain_dns_records dr
JOIN line_groups lg ON dr.owner_id = lg.id
WHERE dr.owner_type = 'line_group'
ORDER BY dr.id DESC
LIMIT 20;

-- 16. Count DNS records by status for node groups
SELECT 
    dr.status,
    COUNT(*) AS count
FROM domain_dns_records dr
WHERE dr.owner_type = 'node_group'
GROUP BY dr.status;

-- 17. Count DNS records by status for line groups
SELECT 
    dr.status,
    COUNT(*) AS count
FROM domain_dns_records dr
WHERE dr.owner_type = 'line_group'
GROUP BY dr.status;

-- 18. Verify node_group_sub_ips mappings
SELECT 
    ngs.id,
    ngs.node_group_id,
    ngs.sub_ip_id,
    ng.name AS node_group_name,
    ns.ip AS sub_ip,
    n.name AS node_name
FROM node_group_sub_ips ngs
JOIN node_groups ng ON ngs.node_group_id = ng.id
JOIN node_sub_ips ns ON ngs.sub_ip_id = ns.id
JOIN nodes n ON ns.node_id = n.id
ORDER BY ngs.id DESC
LIMIT 20;

-- 19. Check for orphaned DNS records (owner_id not exists)
SELECT 
    dr.id,
    dr.type,
    dr.owner_type,
    dr.owner_id,
    dr.status
FROM domain_dns_records dr
WHERE dr.owner_type = 'node_group'
  AND NOT EXISTS (
      SELECT 1 FROM node_groups ng WHERE ng.id = dr.owner_id
  )
UNION ALL
SELECT 
    dr.id,
    dr.type,
    dr.owner_type,
    dr.owner_id,
    dr.status
FROM domain_dns_records dr
WHERE dr.owner_type = 'line_group'
  AND NOT EXISTS (
      SELECT 1 FROM line_groups lg WHERE lg.id = dr.owner_id
  );

-- 20. Statistics summary
SELECT 
    'node_groups' AS table_name,
    COUNT(*) AS total_count,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) AS active_count,
    SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END) AS inactive_count
FROM node_groups
UNION ALL
SELECT 
    'line_groups' AS table_name,
    COUNT(*) AS total_count,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) AS active_count,
    SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END) AS inactive_count
FROM line_groups
UNION ALL
SELECT 
    'node_group_sub_ips' AS table_name,
    COUNT(*) AS total_count,
    0 AS active_count,
    0 AS inactive_count
FROM node_group_sub_ips;
