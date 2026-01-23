-- Create config_versions table
CREATE TABLE IF NOT EXISTS config_versions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  version BIGINT NOT NULL UNIQUE COMMENT '配置版本号（时间戳毫秒）',
  node_id INT NOT NULL COMMENT '节点ID',
  payload TEXT NOT NULL COMMENT '配置payload（JSON）',
  status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/applied/failed',
  applied_at DATETIME COMMENT '应用时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_node_version (node_id, version),
  INDEX idx_created_at (created_at),
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置版本记录';
