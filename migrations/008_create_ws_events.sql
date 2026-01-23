-- 创建 ws_events 表（WebSocket 事件存储）
CREATE TABLE IF NOT EXISTS ws_events (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '事件ID（自增，作为eventId）',
    topic VARCHAR(64) NOT NULL COMMENT '事件主题（如：websites）',
    event_type ENUM('add', 'update', 'delete') NOT NULL COMMENT '事件类型',
    payload JSON NOT NULL COMMENT '事件负载（推送内容，前端可直接用）',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_topic_id (topic, id) COMMENT '按主题和ID查询索引（用于增量补发）'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='WebSocket事件表';
