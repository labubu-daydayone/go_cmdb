-- 012_create_release_task_nodes.sql
-- 创建发布任务节点表

CREATE TABLE IF NOT EXISTS `release_task_nodes` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `release_task_id` bigint NOT NULL COMMENT '发布任务ID',
  `node_id` int NOT NULL COMMENT '节点ID',
  `batch` int NOT NULL DEFAULT 1 COMMENT '批次',
  `status` enum('pending','running','success','failed','skipped') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `error_msg` varchar(255) DEFAULT NULL COMMENT '错误信息',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime DEFAULT NULL COMMENT '完成时间',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_release_task_node` (`release_task_id`,`node_id`),
  KEY `idx_release_task_id` (`release_task_id`),
  KEY `idx_node_id` (`node_id`),
  KEY `idx_batch` (`batch`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布任务节点表';
