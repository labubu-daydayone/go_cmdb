-- T1-04 Websites SQL Verification Script
-- 验证网站管理的数据完整性

-- 1. 查看所有网站
SELECT 'Websites:' AS '';
SELECT id, line_group_id, cache_rule_id, origin_mode, origin_group_id, origin_set_id, 
       redirect_url, redirect_status_code, status, created_at
FROM websites
ORDER BY id;

-- 2. 查看所有网站域名
SELECT 'Website Domains:' AS '';
SELECT id, website_id, domain, is_primary, cname, created_at
FROM website_domains
ORDER BY website_id, id;

-- 3. 查看所有网站HTTPS配置
SELECT 'Website HTTPS:' AS '';
SELECT id, website_id, enabled, force_redirect, hsts, cert_mode, 
       certificate_id, acme_provider_id, acme_account_id, created_at
FROM website_https
ORDER BY website_id;

-- 4. 查看所有origin_sets
SELECT 'Origin Sets:' AS '';
SELECT id, source, origin_group_id, created_at
FROM origin_sets
ORDER BY id;

-- 5. 查看所有origin_addresses
SELECT 'Origin Addresses:' AS '';
SELECT id, origin_set_id, role, protocol, address, weight, enabled, created_at
FROM origin_addresses
ORDER BY origin_set_id, id;

-- 6. 查看网站关联的DNS记录
SELECT 'DNS Records for Websites:' AS '';
SELECT dr.id, dr.domain_id, dr.owner_type, dr.owner_id, dr.type, dr.name, dr.value, 
       dr.ttl, dr.status, dr.last_error
FROM domain_dns_records dr
WHERE dr.owner_type = 'website_domain'
ORDER BY dr.owner_id, dr.id;

-- 7. 网站与line_group的关联
SELECT 'Website-LineGroup Associations:' AS '';
SELECT w.id AS website_id, w.line_group_id, lg.name AS line_group_name, lg.cname
FROM websites w
LEFT JOIN line_groups lg ON w.line_group_id = lg.id
ORDER BY w.id;

-- 8. 网站与origin_group的关联（group模式）
SELECT 'Website-OriginGroup Associations (group mode):' AS '';
SELECT w.id AS website_id, w.origin_mode, w.origin_group_id, og.name AS origin_group_name
FROM websites w
LEFT JOIN origin_groups og ON w.origin_group_id = og.id
WHERE w.origin_mode = 'group'
ORDER BY w.id;

-- 9. 网站与origin_set的关联
SELECT 'Website-OriginSet Associations:' AS '';
SELECT w.id AS website_id, w.origin_mode, w.origin_set_id, os.source, os.origin_group_id
FROM websites w
LEFT JOIN origin_sets os ON w.origin_set_id = os.id
WHERE w.origin_set_id IS NOT NULL AND w.origin_set_id > 0
ORDER BY w.id;

-- 10. 网站域名统计
SELECT 'Website Domain Statistics:' AS '';
SELECT w.id AS website_id, COUNT(wd.id) AS domain_count, 
       GROUP_CONCAT(wd.domain ORDER BY wd.is_primary DESC SEPARATOR ', ') AS domains
FROM websites w
LEFT JOIN website_domains wd ON w.website_id = wd.website_id
GROUP BY w.id
ORDER BY w.id;

-- 11. Origin mode统计
SELECT 'Origin Mode Statistics:' AS '';
SELECT origin_mode, COUNT(*) AS count
FROM websites
GROUP BY origin_mode;

-- 12. HTTPS启用统计
SELECT 'HTTPS Statistics:' AS '';
SELECT 
  COUNT(*) AS total_websites,
  SUM(CASE WHEN wh.enabled = 1 THEN 1 ELSE 0 END) AS https_enabled,
  SUM(CASE WHEN wh.enabled = 0 OR wh.id IS NULL THEN 1 ELSE 0 END) AS https_disabled
FROM websites w
LEFT JOIN website_https wh ON w.id = wh.website_id;

-- 13. 验证数据完整性
SELECT 'Data Integrity Check:' AS '';
SELECT 
  'Orphan website_domains' AS check_name,
  COUNT(*) AS count
FROM website_domains wd
WHERE NOT EXISTS (SELECT 1 FROM websites w WHERE w.id = wd.website_id)
UNION ALL
SELECT 
  'Orphan website_https' AS check_name,
  COUNT(*) AS count
FROM website_https wh
WHERE NOT EXISTS (SELECT 1 FROM websites w WHERE w.id = wh.website_id)
UNION ALL
SELECT 
  'Orphan origin_addresses' AS check_name,
  COUNT(*) AS count
FROM origin_addresses oa
WHERE NOT EXISTS (SELECT 1 FROM origin_sets os WHERE os.id = oa.origin_set_id)
UNION ALL
SELECT 
  'Websites with invalid line_group_id' AS check_name,
  COUNT(*) AS count
FROM websites w
WHERE NOT EXISTS (SELECT 1 FROM line_groups lg WHERE lg.id = w.line_group_id);
