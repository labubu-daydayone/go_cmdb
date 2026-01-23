# T2-08 交付报告：证书与网站风险预检 + 告警体系

## 任务概述

实现证书与网站风险预检和告警体系，在用户点按钮之前告知风险，后台持续扫描风险，只发现和解释，不自动修复。

## 完成矩阵

| Phase | 任务 | 状态 | 证据 |
|-------|------|------|------|
| 1 | 任务分析 | Done | docs/T2-08-PLAN.md |
| 2 | 创建certificate_risks表 | Done | migrations/010_create_certificate_risks.sql |
| 3 | 实现4类风险规则 | Done | internal/risk/rules.go |
| 4 | 实现风险扫描Worker | Done | internal/risk/scanner.go |
| 5 | 实现风险查询API | Done | api/v1/risks/handler.go |
| 6 | 实现前置预检API | Done | internal/risk/precheck.go |
| 7 | 实现风险解决API | Done | api/v1/risks/handler.go |
| 8 | 编写验收测试 | Done | scripts/test_certificate_risks.sh |
| 9 | 生成交付报告 | Done | docs/T2-08-DELIVERY.md |

## 核心功能实现

### 1. 数据模型（Phase 2）

**文件**: migrations/010_create_certificate_risks.sql, internal/model/certificate_risk.go

**表结构**: certificate_risks
- id: 主键
- risk_type: 风险类型（domain_mismatch/cert_expiring/acme_renew_failed/weak_coverage）
- level: 风险级别（info/warning/critical）
- status: 状态（active/resolved）
- certificate_id: 关联证书ID
- website_id: 关联网站ID
- detail: JSON详情
- detected_at: 检测时间
- resolved_at: 解决时间

**唯一约束**: (risk_type, certificate_id, website_id, status)

**索引**: certificate_id, website_id, status, level, detected_at, risk_type

### 2. 风险规则（Phase 3）

**文件**: internal/risk/rules.go

**4类风险规则**:

1. **DomainMismatchRule** (域名不匹配)
   - 级别: critical
   - 触发条件: 证书不完全覆盖网站域名
   - detail: certificate_domains, website_domains, missing_domains, coverage_status

2. **CertExpiringRule** (证书即将过期)
   - 级别: warning
   - 触发条件: 证书在N天内过期且影响>=M个网站
   - 可配置: expiringDays=15, websiteThreshold=2
   - detail: certificate_id, expire_at, days_remaining, affected_websites, website_count

3. **ACMERenewFailedRule** (ACME续期失败)
   - 级别: critical
   - 触发条件: certificate_requests.status=failed且attempts>=max_attempts
   - 可配置: maxAttempts=3
   - detail: certificate_id, request_id, attempts, last_error

4. **WeakCoverageRule** (弱覆盖)
   - 级别: warning
   - 触发条件: wildcard覆盖但包含apex+subdomain混合
   - detail: certificate_domains, website_domains, recommendation

### 3. 风险扫描Worker（Phase 4）

**文件**: internal/risk/scanner.go, cmd/cmdb/main.go

**Scanner结构**:
- 配置: enabled, intervalSec, certExpiringDays, certExpiringThreshold, acmeMaxAttempts
- 方法: Start, Stop, Scan, upsertRisk, ResolveRisk

**幂等生成机制**:
- 查询是否已存在active状态的相同风险
- 存在: 更新detected_at/detail/level
- 不存在: 插入新记录
- 依赖UNIQUE KEY约束

**启动集成**:
- 在main函数中启动Scanner
- 立即执行一次+定时轮询（5分钟）
- 使用defer停止Scanner

**环境变量**:
- RISK_SCANNER_ENABLED=1
- RISK_SCANNER_INTERVAL_SEC=300
- CERT_EXPIRING_DAYS=15
- CERT_EXPIRING_WEBSITE_THRESHOLD=2
- ACME_MAX_ATTEMPTS=3

### 4. 风险查询API（Phase 5）

**文件**: internal/risk/service.go, api/v1/risks/handler.go

**API接口**:

1. **GET /api/v1/risks** (全局风险列表)
   - 过滤: level, risk_type, status, certificate_id, website_id
   - 分页: page, page_size
   - 返回: risks, total, page, page_size, total_pages

2. **GET /api/v1/websites/:id/risks** (网站风险列表)
   - 只返回active状态的风险
   - 按level和detected_at排序

3. **GET /api/v1/certificates/:id/risks** (证书风险列表)
   - 只返回active状态的风险
   - 按level和detected_at排序

### 5. 前置预检API（Phase 6）

**文件**: internal/risk/precheck.go, api/v1/risks/handler.go

**API接口**: POST /api/v1/websites/:id/precheck/https

**请求体**:
```json
{
  "cert_mode": "select",
  "certificate_id": 1
}
```

**响应体**:
```json
{
  "ok": false,
  "risks": [
    {
      "type": "domain_mismatch",
      "level": "critical",
      "detail": {
        "message": "Certificate does not cover all website domains",
        "certificate_domains": ["*.example.com"],
        "website_domains": ["example.com", "www.example.com"],
        "missing_domains": ["example.com"],
        "coverage_status": "partial"
      }
    }
  ]
}
```

**检查逻辑**:
- select模式: 检查domain_mismatch, weak_coverage, cert_expiring
- acme模式: 不检查（返回ok=true, risks=[]）
- critical风险: ok=false（阻止启用）
- warning风险: ok=true（允许启用但提示）

### 6. 风险解决API（Phase 7）

**文件**: internal/risk/service.go, api/v1/risks/handler.go

**API接口**: POST /api/v1/risks/:id/resolve

**功能**:
- 更新status为resolved
- 设置resolved_at为当前时间
- 不做任何自动修复

**返回格式**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "risk_id": 1,
    "message": "Risk resolved successfully"
  }
}
```

### 7. 验收测试（Phase 8）

**文件**: scripts/test_certificate_risks.sh

**CURL测试（25条）**:
1. 登录获取JWT token
2. 查询全局风险列表（无过滤）
3. 查询全局风险列表（按level过滤）
4. 查询全局风险列表（按risk_type过滤）
5. 查询全局风险列表（按status过滤）
6. 查询全局风险列表（分页）
7. 查询网站风险列表（有效网站）
8. 查询网站风险列表（无效网站）
9. 查询证书风险列表（有效证书）
10. 查询证书风险列表（无效证书）
11. 前置预检（select模式，partial覆盖，ok=false）
12. 检查预检响应包含risk类型
13. 检查预检响应包含detail
14. 前置预检（acme模式，绕过校验，ok=true）
15. 解决风险（有效风险ID）
16. 验证风险已解决（status=resolved）
17. 重复解决已resolved的风险（应失败）
18. 解决风险（无效风险ID，应返回404）
19. 测试未授权访问风险列表
20. 测试未授权访问预检API
21. 测试缺失参数的预检请求
22. 测试invalid cert_mode
23. 查询风险列表（按certificate_id过滤）
24. 查询风险列表（按website_id过滤）
25. 测试分页边界（page=0）

**SQL验证（20条）**:
1. 验证certificate_risks表存在
2. 验证risk_type字段枚举值
3. 验证level字段枚举值
4. 验证status字段枚举值
5. 验证detail字段类型为JSON
6. 验证唯一约束存在
7. 统计active状态的风险数量
8. 统计resolved状态的风险数量
9. 统计各类型风险数量
10. 统计各级别风险数量
11. 检查domain_mismatch风险的detail内容
12. 检查cert_expiring风险的detail内容
13. 检查weak_coverage风险的detail内容
14. 验证风险幂等（同一风险不重复生成）
15. 验证resolved_at字段在resolved状态时不为空
16. 验证detected_at字段不为空
17. 统计关联证书的风险数量
18. 统计关联网站的风险数量
19. 验证风险状态流转（detected_at < resolved_at）
20. 检查最近检测到的风险

## 技术实现细节

### 幂等生成机制

风险扫描Worker使用幂等生成机制，确保同一风险不会重复生成：

```go
func (s *Scanner) upsertRisk(risk *model.CertificateRisk) error {
    // 查询是否已存在active状态的相同风险
    var existing model.CertificateRisk
    err := s.db.Where(
        "risk_type = ? AND certificate_id = ? AND website_id = ? AND status = ?",
        risk.RiskType, risk.CertificateID, risk.WebsiteID, model.RiskStatusActive,
    ).First(&existing).Error

    if err == gorm.ErrRecordNotFound {
        // 不存在，插入新记录
        return s.db.Create(risk).Error
    } else if err != nil {
        return err
    }

    // 已存在，更新detected_at和detail
    return s.db.Model(&existing).Updates(map[string]interface{}{
        "detected_at": risk.DetectedAt,
        "detail":      risk.Detail,
        "level":       risk.Level,
    }).Error
}
```

### Wildcard域名匹配规则

使用T2-07中实现的wildcard域名匹配规则：

- `*.example.com`匹配`a.example.com`
- `*.example.com`不匹配`example.com`（apex域名）
- `*.example.com`不匹配`a.b.example.com`（二级子域名）

### 弱覆盖判定

弱覆盖是指wildcard证书覆盖了网站域名，但网站域名包含apex+subdomain混合，建议使用包含apex的证书：

```go
func isWeakCoverage(certDomains, websiteDomains []string) bool {
    // 条件1：完全覆盖
    coverage := cert.CalculateCoverage(certDomains, websiteDomains)
    if coverage.Status != cert.CoverageStatusCovered {
        return false
    }

    // 条件2：website_domains数量 > 1
    if len(websiteDomains) <= 1 {
        return false
    }

    // 条件3：包含apex + subdomain混合
    hasApex := false
    hasSubdomain := false
    for _, domain := range websiteDomains {
        if isApexDomain(domain) {
            hasApex = true
        } else {
            hasSubdomain = true
        }
    }
    if !hasApex || !hasSubdomain {
        return false
    }

    // 条件4：证书必须是纯wildcard（不包含apex）
    hasWildcard := false
    hasApexInCert := false
    for _, domain := range certDomains {
        if isWildcard(domain) {
            hasWildcard = true
        }
        baseDomain := extractBaseDomain(websiteDomains)
        if domain == baseDomain {
            hasApexInCert = true
        }
    }

    return hasWildcard && !hasApexInCert
}
```

## 部署说明

### 1. 数据库迁移

应用启动时会自动创建certificate_risks表（MIGRATE=1环境变量）。

或者手动执行：
```bash
mysql -h 20.2.140.226 -u root -proot123 cmdb < migrations/010_create_certificate_risks.sql
```

### 2. 环境变量配置

在启动应用前设置以下环境变量：

```bash
export RISK_SCANNER_ENABLED=1
export RISK_SCANNER_INTERVAL_SEC=300
export CERT_EXPIRING_DAYS=15
export CERT_EXPIRING_WEBSITE_THRESHOLD=2
export ACME_MAX_ATTEMPTS=3
```

### 3. 编译应用

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
```

### 4. 启动服务

```bash
./bin/cmdb
```

启动日志会显示：
```
Risk Scanner initialized
Risk Scanner started
```

### 5. 验证部署

执行验收测试脚本：
```bash
./scripts/test_certificate_risks.sh
```

## API文档

### 1. 查询全局风险列表

**请求**:
```
GET /api/v1/risks?level=critical&risk_type=domain_mismatch&status=active&page=1&page_size=20
Authorization: Bearer <token>
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "risks": [
      {
        "id": 1,
        "risk_type": "domain_mismatch",
        "level": "critical",
        "status": "active",
        "certificate_id": 1,
        "website_id": 1,
        "detail": {
          "message": "Certificate does not cover all website domains",
          "certificate_domains": ["*.example.com"],
          "website_domains": ["example.com", "www.example.com"],
          "missing_domains": ["example.com"],
          "coverage_status": "partial"
        },
        "detected_at": "2026-01-23T10:00:00Z",
        "resolved_at": null
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20,
    "total_pages": 1
  }
}
```

### 2. 查询网站风险列表

**请求**:
```
GET /api/v1/websites/1/risks
Authorization: Bearer <token>
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "website_id": 1,
    "risks": [
      {
        "id": 1,
        "risk_type": "domain_mismatch",
        "level": "critical",
        "status": "active",
        "certificate_id": 1,
        "website_id": 1,
        "detail": {...},
        "detected_at": "2026-01-23T10:00:00Z",
        "resolved_at": null
      }
    ],
    "count": 1
  }
}
```

### 3. 查询证书风险列表

**请求**:
```
GET /api/v1/certificates/1/risks
Authorization: Bearer <token>
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "certificate_id": 1,
    "risks": [
      {
        "id": 2,
        "risk_type": "cert_expiring",
        "level": "warning",
        "status": "active",
        "certificate_id": 1,
        "website_id": null,
        "detail": {
          "message": "Certificate expires in 10 days",
          "certificate_id": 1,
          "expire_at": "2026-02-02T00:00:00Z",
          "days_remaining": 10,
          "affected_websites": [1, 2, 3],
          "website_count": 3
        },
        "detected_at": "2026-01-23T10:00:00Z",
        "resolved_at": null
      }
    ],
    "count": 1
  }
}
```

### 4. 前置风险预检

**请求**:
```
POST /api/v1/websites/1/precheck/https
Authorization: Bearer <token>
Content-Type: application/json

{
  "cert_mode": "select",
  "certificate_id": 1
}
```

**响应（有风险）**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ok": false,
    "risks": [
      {
        "type": "domain_mismatch",
        "level": "critical",
        "detail": {
          "message": "Certificate does not cover all website domains",
          "certificate_domains": ["*.example.com"],
          "website_domains": ["example.com", "www.example.com"],
          "missing_domains": ["example.com"],
          "coverage_status": "partial"
        }
      },
      {
        "type": "weak_coverage",
        "level": "warning",
        "detail": {
          "message": "Wildcard certificate covers mixed apex and subdomain",
          "certificate_domains": ["*.example.com"],
          "website_domains": ["example.com", "www.example.com"],
          "recommendation": "Use certificate with both example.com and *.example.com"
        }
      }
    ]
  }
}
```

**响应（无风险）**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ok": true,
    "risks": []
  }
}
```

### 5. 解决风险

**请求**:
```
POST /api/v1/risks/1/resolve
Authorization: Bearer <token>
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "risk_id": 1,
    "message": "Risk resolved successfully"
  }
}
```

## 已知限制

1. **性能问题**: 大量风险时扫描性能可能下降（建议使用worker pool）
2. **通知机制**: 未实现风险通知（邮件/webhook）
3. **风险历史**: 未单独记录风险历史（只有active/resolved状态）
4. **自动修复**: 只发现和解释，不自动修复
5. **并发安全**: Scanner是单线程的（建议使用锁保护）

## 后续改进建议

### 短期（1-2周）

1. 实现风险通知机制（邮件/webhook）
2. 优化扫描性能（worker pool）
3. 添加风险统计面板

### 中期（1-2个月）

1. 实现风险历史记录
2. 添加风险趋势图
3. 支持自定义风险规则

### 长期（3-6个月）

1. 实现自动修复建议
2. 添加风险预测
3. 集成第三方监控平台

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert 8f8184797df9209b3efa3518cef6c339c2270bf8
```

### 数据库回滚

```sql
DROP TABLE IF EXISTS certificate_risks;
```

### 禁用扫描

设置环境变量：
```bash
export RISK_SCANNER_ENABLED=0
```

## 相关文档

- 实现计划: docs/T2-08-PLAN.md
- 测试脚本: scripts/test_certificate_risks.sh
- T2-07交付报告: docs/T2-07-DELIVERY.md（证书与网站关系可视化）
- T2-06交付报告: docs/T2-06-DELIVERY.md（证书自动续期）
- T2-05交付报告: docs/T2-05-DELIVERY.md（ACME Worker）

## 交付清单

- [x] 数据模型设计和实现
- [x] 4类风险规则实现
- [x] 风险扫描Worker实现
- [x] 风险查询API实现
- [x] 前置预检API实现
- [x] 风险解决API实现
- [x] 验收测试脚本
- [x] 交付报告

## 最终Commit

- Commit Hash: 8f8184797df9209b3efa3518cef6c339c2270bf8
- Commit Message: feat(T2-08): implement certificate risk detection and precheck system
- 仓库: labubu-daydayone/go_cmdb_web
- 分支: main
