-- 011_create_release_tasks.sql
-- 创建发布任务表

CREATE TABLE IF NOT EXISTS `release_tasks` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type` enum('apply_config') NOT NULL COMMENT '任务类型',
  `target` enum('cdn') NOT NULL COMMENT '目标类型',
  `version` bigint NOT NULL COMMENT '版本号',
  `status` enum('pending','running','success','failed','paused') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `total_nodes` int NOT NULL DEFAULT 0 COMMENT '总节点数',
  `success_nodes` int NOT NULL DEFAULT 0 COMMENT '成功节点数',
  `failed_nodes` int NOT NULL DEFAULT 0 COMMENT '失败节点数',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布任务表';
