# T2-08 证书与网站风险预检 + 告警体系 - 实现计划

## 任务概述

任务编号：T2-08
优先级：P0
前置依赖：T2-07已通过

核心原则：**只做"风险发现 + 解释"，不自动修复**

## 核心目标

在用户点按钮之前，告诉他：
1. 会不会失败
2. 风险在哪里
3. 影响范围有多大

并且让系统在后台持续扫描这些风险。

## 必须解决的4类风险

### 风险1：domain_mismatch（域名不匹配）

触发条件：
- website_domains新增/删除
- website_https.enabled = 1
- cert_mode = select
- certificate_domains不再100%覆盖

结果行为：
- 不自动解绑
- 生成一条risk记录
- 在查询接口中可见

### 风险2：cert_expiring（证书即将过期）

触发条件：
- certificates.expire_at < now + N天（默认15，可env）
- certificate_bindings.active = 1
- 绑定网站数量 >= 2（阈值可配置）

结果行为：
- 生成风险
- 风险信息必须包含：
  - 证书id
  - 剩余天数
  - 受影响网站列表

### 风险3：acme_renew_failed（ACME续期失败）

触发条件：
- certificate_requests.status = failed
- attempts >= max_attempts
- 对应certificate当前仍被active绑定

结果行为：
- 生成风险
- 标记为high

### 风险4：weak_coverage（弱覆盖）

定义：
- wildcard覆盖成立
- 但website_domains数量 > 1
- 且包含apex + subdomain混合

例如：
- 网站域名：example.com, www.example.com
- 证书：*.example.com

结果：
- covered（T2-07已允许）
- 但在T2-08中必须产生warning级风险

目的：提醒运营"这不是最佳证书"

## 数据模型

### certificate_risks表（新增）

```sql
CREATE TABLE certificate_risks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    risk_type ENUM('domain_mismatch', 'cert_expiring', 'acme_renew_failed', 'weak_coverage') NOT NULL,
    level ENUM('info', 'warning', 'critical') NOT NULL,
    certificate_id INT NULL,
    website_id INT NULL,
    detail JSON NOT NULL COMMENT '必须包含人类可解释信息',
    status ENUM('active', 'resolved') NOT NULL DEFAULT 'active',
    detected_at DATETIME NOT NULL,
    resolved_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_risk (risk_type, certificate_id, website_id, status),
    INDEX idx_certificate_id (certificate_id),
    INDEX idx_website_id (website_id),
    INDEX idx_status (status),
    INDEX idx_level (level),
    INDEX idx_detected_at (detected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='证书与网站风险记录';
```

约束：
- 同一(risk_type, certificate_id, website_id, status=active)只能有一条

## 必须实现的功能点

### 1. 风险扫描器（Worker）

启动方式：main.go启动goroutine
扫描周期：默认5分钟（env可配）

行为：
- 扫描所有active website + certificate
- 幂等生成risk（不能刷爆表）

### 2. 风险查询API

#### 2.1 全局风险列表
GET /api/v1/risks

支持query：
- level
- risk_type
- status
- certificate_id
- website_id

#### 2.2 网站视角风险
GET /api/v1/websites/{id}/risks

#### 2.3 证书视角风险
GET /api/v1/certificates/{id}/risks

### 3. 风险解决标记（人工）

POST /api/v1/risks/{id}/resolve

行为：
- status -> resolved
- 写resolved_at
- 不做任何自动修复

### 4. 前置风险预检API

POST /api/v1/websites/{id}/precheck/https

输入：
- cert_mode
- certificate_id（如果select）

输出：
- ok: true/false
- risks: []（与certificate_risks.detail同结构）

用途：
- 前端"启用HTTPS"前调用
- 即使ok=true，也可能返回warning风险

## 必须覆盖的场景

1. 新增网站域名 → 产生domain_mismatch风险
2. 一个证书绑定3个网站 + 10天后过期 → cert_expiring
3. ACME连续失败 → acme_renew_failed
4. *.example.com + apex + www → weak_coverage
5. resolve后风险不再出现在active列表
6. 再次扫描不会重复生成同一风险

## 验收要求

1. go test ./...
2. curl不少于20条，必须包含：
   - 风险扫描前后对比
   - 预检API（ok=true + warning）
   - critical风险阻断HTTPS enable（结合T2-07）
   - resolve风险
3. SQL不少于15条，必须包含：
   - risk幂等
   - risk状态流转
   - detail内容校验

## 回滚策略

- git revert本任务提交
- 停止risk worker
- 不影响证书/网站/agent

## 实现阶段

### Phase 1: 任务分析
- 创建T2-08-PLAN.md
- 明确4类风险规则
- 设计API接口

### Phase 2: 数据模型
- 创建certificate_risks表
- 实现model层
- 添加唯一约束

### Phase 3: 风险规则逻辑
- 实现domain_mismatch检测
- 实现cert_expiring检测
- 实现acme_renew_failed检测
- 实现weak_coverage检测

### Phase 4: 风险扫描Worker
- 实现RiskScanner
- 幂等生成risk
- 5分钟轮询
- 在main.go中启动

### Phase 5: 风险查询API
- GET /api/v1/risks（全局列表）
- GET /api/v1/websites/{id}/risks（网站视角）
- GET /api/v1/certificates/{id}/risks（证书视角）

### Phase 6: 前置预检API
- POST /api/v1/websites/{id}/precheck/https
- 返回ok + risks
- 支持warning级风险

### Phase 7: 风险解决API
- POST /api/v1/risks/{id}/resolve
- 更新status和resolved_at
- 不做自动修复

### Phase 8: 验收测试
- 20+条curl测试
- 15+条SQL验证
- 覆盖所有场景

### Phase 9: 交付报告
- 生成T2-08-DELIVERY.md
- 生成T2-08-SUMMARY.md
- 提交代码到GitHub

## 关键技术点

### 1. 风险幂等生成

使用UNIQUE KEY约束：
```sql
UNIQUE KEY uk_risk (risk_type, certificate_id, website_id, status)
```

INSERT时使用ON DUPLICATE KEY UPDATE：
```sql
INSERT INTO certificate_risks (...) VALUES (...)
ON DUPLICATE KEY UPDATE detected_at = NOW()
```

### 2. 风险级别判定

- domain_mismatch: critical
- cert_expiring: warning（默认15天）
- acme_renew_failed: critical
- weak_coverage: warning

### 3. 风险detail格式

domain_mismatch:
```json
{
  "message": "Certificate does not cover all website domains",
  "certificate_domains": ["*.example.com"],
  "website_domains": ["example.com", "www.example.com"],
  "missing_domains": ["example.com"]
}
```

cert_expiring:
```json
{
  "message": "Certificate expires in 10 days and affects 3 websites",
  "certificate_id": 1,
  "expire_at": "2026-02-02T00:00:00Z",
  "days_remaining": 10,
  "affected_websites": [1, 2, 3]
}
```

acme_renew_failed:
```json
{
  "message": "ACME renewal failed after 3 attempts",
  "certificate_id": 1,
  "request_id": 123,
  "attempts": 3,
  "last_error": "DNS validation failed"
}
```

weak_coverage:
```json
{
  "message": "Wildcard certificate covers mixed apex and subdomain",
  "certificate_domains": ["*.example.com"],
  "website_domains": ["example.com", "www.example.com"],
  "recommendation": "Use certificate with both example.com and *.example.com"
}
```

### 4. 前置预检逻辑

```go
func (s *RiskService) PrecheckHTTPS(websiteID int, certMode string, certificateID int) (*PrecheckResult, error) {
    risks := []Risk{}
    
    if certMode == "select" {
        // 检查domain_mismatch
        if !isCovered(certificateID, websiteID) {
            risks = append(risks, Risk{
                Type: "domain_mismatch",
                Level: "critical",
                Detail: ...
            })
        }
        
        // 检查weak_coverage
        if isWeakCoverage(certificateID, websiteID) {
            risks = append(risks, Risk{
                Type: "weak_coverage",
                Level: "warning",
                Detail: ...
            })
        }
        
        // 检查cert_expiring
        if isExpiring(certificateID) {
            risks = append(risks, Risk{
                Type: "cert_expiring",
                Level: "warning",
                Detail: ...
            })
        }
    }
    
    // 有critical风险时，ok=false
    ok := true
    for _, r := range risks {
        if r.Level == "critical" {
            ok = false
            break
        }
    }
    
    return &PrecheckResult{
        OK: ok,
        Risks: risks,
    }, nil
}
```

## 环境变量

```bash
# 风险扫描器配置
RISK_SCANNER_ENABLED=true
RISK_SCANNER_INTERVAL_SEC=300  # 5分钟

# 证书过期预警天数
CERT_EXPIRING_DAYS=15

# 证书过期影响网站数量阈值
CERT_EXPIRING_WEBSITE_THRESHOLD=2
```

## 文件结构

```
internal/
  risk/
    scanner.go          # 风险扫描器
    rules.go            # 4类风险规则
    service.go          # 风险服务层
api/v1/
  risks/
    handler.go          # 风险API Handler
migrations/
  010_create_certificate_risks.sql  # 数据库迁移
scripts/
  test_certificate_risks.sh         # 验收测试脚本
docs/
  T2-08-PLAN.md         # 本文档
  T2-08-DELIVERY.md     # 交付报告
  T2-08-SUMMARY.md      # 交付总结
```

## 重要一句话

**T2-07是"防止犯错"**
**T2-08是"提前告诉你要犯错"**

这是一个成熟控制系统和"能跑系统"的分水岭。
