-- 025_create_origin_sets_and_items.sql

-- 修改 origin_sets 表，添加缺失字段
ALTER TABLE origin_sets
ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '' AFTER id,
ADD COLUMN description TEXT NULL AFTER name,
ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'active' AFTER description;

-- 创建 origin_set_items 表
CREATE TABLE IF NOT EXISTS origin_set_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    origin_set_id BIGINT UNSIGNED NOT NULL,
    origin_group_id BIGINT UNSIGNED NOT NULL,
    snapshot_json JSON NOT NULL COMMENT '冻结的回源地址快照',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_origin_set_id (origin_set_id),
    KEY idx_origin_group_id (origin_group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回源快照项表';
