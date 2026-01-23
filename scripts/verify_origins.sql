-- T1-03 回源分组与网站回源快照 SQL验证脚本

USE go_cmdb;

-- 1. 验证origin_groups表结构
SELECT '1. origin_groups表结构' AS test_name;
SHOW CREATE TABLE origin_groups\G

-- 2. 验证origin_group_addresses表结构
SELECT '2. origin_group_addresses表结构' AS test_name;
SHOW CREATE TABLE origin_group_addresses\G

-- 3. 验证origin_sets表结构
SELECT '3. origin_sets表结构' AS test_name;
SHOW CREATE TABLE origin_sets\G

-- 4. 验证origin_addresses表结构
SELECT '4. origin_addresses表结构' AS test_name;
SHOW CREATE TABLE origin_addresses\G

-- 5. 验证websites表结构（回源字段）
SELECT '5. websites表回源字段' AS test_name;
DESC websites;

-- 6. 验证origin_groups数据
SELECT '6. origin_groups数据' AS test_name;
SELECT id, name, description, status, created_at
FROM origin_groups
ORDER BY id DESC
LIMIT 5;

-- 7. 验证origin_group_addresses数据
SELECT '7. origin_group_addresses数据' AS test_name;
SELECT id, origin_group_id, role, protocol, address, weight, enabled
FROM origin_group_addresses
ORDER BY id DESC
LIMIT 10;

-- 8. 验证origin_sets数据
SELECT '8. origin_sets数据' AS test_name;
SELECT id, source, origin_group_id, created_at
FROM origin_sets
ORDER BY id DESC
LIMIT 5;

-- 9. 验证origin_addresses数据
SELECT '9. origin_addresses数据' AS test_name;
SELECT id, origin_set_id, role, protocol, address, weight, enabled
FROM origin_addresses
ORDER BY id DESC
LIMIT 10;

-- 10. 验证websites与origin_sets的关联
SELECT '10. websites与origin_sets关联' AS test_name;
SELECT w.id, w.domain, w.origin_mode, w.origin_group_id, w.origin_set_id,
       os.source AS set_source, os.origin_group_id AS set_group_id
FROM websites w
LEFT JOIN origin_sets os ON w.origin_set_id = os.id
WHERE w.origin_set_id > 0
ORDER BY w.id DESC
LIMIT 5;

-- 11. 验证快照与分组的隔离（同一个group_id可以被多个set引用）
SELECT '11. 快照与分组隔离验证' AS test_name;
SELECT origin_group_id, COUNT(*) AS set_count
FROM origin_sets
WHERE origin_group_id > 0
GROUP BY origin_group_id
HAVING set_count > 1;

-- 12. 验证相同IP在不同set中的权重
SELECT '12. 相同IP不同set权重' AS test_name;
SELECT address, origin_set_id, weight, enabled
FROM origin_addresses
WHERE address IN (
    SELECT address
    FROM origin_addresses
    GROUP BY address
    HAVING COUNT(DISTINCT origin_set_id) > 1
)
ORDER BY address, origin_set_id;

-- 13. 验证origin_group被website引用的情况
SELECT '13. origin_group引用情况' AS test_name;
SELECT og.id, og.name, COUNT(w.id) AS website_count
FROM origin_groups og
LEFT JOIN websites w ON og.id = w.origin_group_id
GROUP BY og.id, og.name
ORDER BY website_count DESC, og.id DESC
LIMIT 5;

-- 14. 验证origin_set的唯一性（每个set只能属于一个website）
SELECT '14. origin_set唯一性验证' AS test_name;
SELECT origin_set_id, COUNT(*) AS website_count
FROM websites
WHERE origin_set_id > 0
GROUP BY origin_set_id
HAVING website_count > 1;

-- 15. 验证group模式的业务约束
SELECT '15. group模式约束验证' AS test_name;
SELECT w.id, w.domain, w.origin_mode, w.origin_group_id, w.origin_set_id,
       os.source
FROM websites w
LEFT JOIN origin_sets os ON w.origin_set_id = os.id
WHERE w.origin_mode = 'group'
  AND (w.origin_group_id = 0 OR w.origin_set_id = 0 OR os.source != 'group');

-- 16. 验证manual模式的业务约束
SELECT '16. manual模式约束验证' AS test_name;
SELECT w.id, w.domain, w.origin_mode, w.origin_group_id, w.origin_set_id,
       os.source
FROM websites w
LEFT JOIN origin_sets os ON w.origin_set_id = os.id
WHERE w.origin_mode = 'manual'
  AND (w.origin_group_id != 0 OR w.origin_set_id = 0 OR os.source != 'manual');

-- 17. 验证redirect模式的业务约束
SELECT '17. redirect模式约束验证' AS test_name;
SELECT id, domain, origin_mode, origin_group_id, origin_set_id, redirect_url, redirect_status_code
FROM websites
WHERE origin_mode = 'redirect'
  AND (origin_group_id != 0 OR origin_set_id != 0);

-- 18. 检查孤儿origin_addresses（origin_set不存在）
SELECT '18. 孤儿origin_addresses检查' AS test_name;
SELECT oa.*
FROM origin_addresses oa
LEFT JOIN origin_sets os ON oa.origin_set_id = os.id
WHERE os.id IS NULL;

-- 19. 检查孤儿origin_group_addresses（origin_group不存在）
SELECT '19. 孤儿origin_group_addresses检查' AS test_name;
SELECT oga.*
FROM origin_group_addresses oga
LEFT JOIN origin_groups og ON oga.origin_group_id = og.id
WHERE og.id IS NULL;

-- 20. 统计摘要
SELECT '20. 统计摘要' AS test_name;
SELECT 
    (SELECT COUNT(*) FROM origin_groups) AS origin_groups_count,
    (SELECT COUNT(*) FROM origin_group_addresses) AS origin_group_addresses_count,
    (SELECT COUNT(*) FROM origin_sets) AS origin_sets_count,
    (SELECT COUNT(*) FROM origin_addresses) AS origin_addresses_count,
    (SELECT COUNT(*) FROM websites WHERE origin_mode = 'group') AS group_mode_websites,
    (SELECT COUNT(*) FROM websites WHERE origin_mode = 'manual') AS manual_mode_websites,
    (SELECT COUNT(*) FROM websites WHERE origin_mode = 'redirect') AS redirect_mode_websites;
