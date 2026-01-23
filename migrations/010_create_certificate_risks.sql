-- 010_create_certificate_risks.sql
-- T2-08: 证书与网站风险预检 + 告警体系

CREATE TABLE IF NOT EXISTS certificate_risks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    risk_type ENUM('domain_mismatch', 'cert_expiring', 'acme_renew_failed', 'weak_coverage') NOT NULL COMMENT '风险类型',
    level ENUM('info', 'warning', 'critical') NOT NULL COMMENT '风险级别',
    certificate_id INT NULL COMMENT '关联证书ID',
    website_id INT NULL COMMENT '关联网站ID',
    detail JSON NOT NULL COMMENT '风险详情（必须包含人类可解释信息）',
    status ENUM('active', 'resolved') NOT NULL DEFAULT 'active' COMMENT '风险状态',
    detected_at DATETIME NOT NULL COMMENT '检测时间',
    resolved_at DATETIME NULL COMMENT '解决时间',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- 唯一约束：同一(risk_type, certificate_id, website_id, status=active)只能有一条
    UNIQUE KEY uk_risk (risk_type, certificate_id, website_id, status),
    
    -- 索引
    INDEX idx_certificate_id (certificate_id),
    INDEX idx_website_id (website_id),
    INDEX idx_status (status),
    INDEX idx_level (level),
    INDEX idx_detected_at (detected_at),
    INDEX idx_risk_type (risk_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='证书与网站风险记录';
