-- 缓存规则组表
CREATE TABLE IF NOT EXISTS `cache_rules` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_cache_rules_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 缓存规则项表
CREATE TABLE IF NOT EXISTS `cache_rule_items` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `cache_rule_id` bigint NOT NULL,
  `match_type` enum('path','suffix','exact') COLLATE utf8mb4_unicode_ci NOT NULL,
  `match_value` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ttl_seconds` bigint NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cache_rule_item_unique` (`cache_rule_id`,`match_type`,`match_value`),
  KEY `idx_cache_rule_items_cache_rule_id` (`cache_rule_id`),
  CONSTRAINT `fk_cache_rule_items_cache_rule` FOREIGN KEY (`cache_rule_id`) REFERENCES `cache_rules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
