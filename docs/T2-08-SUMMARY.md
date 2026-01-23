# T2-08 交付总结：证书与网站风险预检 + 告警体系

## 核心目标

在用户点按钮之前告知风险，后台持续扫描风险，只发现和解释，不自动修复。

## 完成情况

9个Phase全部完成，45个测试用例（25条CURL + 20条SQL）。

## 核心功能

### 1. 4类风险规则

1. **domain_mismatch** (critical): 证书不完全覆盖网站域名
2. **cert_expiring** (warning): 证书即将过期且影响多个网站
3. **acme_renew_failed** (critical): ACME续期连续失败
4. **weak_coverage** (warning): wildcard覆盖但包含apex+subdomain混合

### 2. 风险扫描Worker

- 5分钟轮询
- 幂等生成机制
- 立即执行一次+定时轮询
- 环境变量配置

### 3. 风险查询API

- GET /api/v1/risks（全局列表，支持过滤和分页）
- GET /api/v1/websites/:id/risks（网站视角）
- GET /api/v1/certificates/:id/risks（证书视角）

### 4. 前置预检API

- POST /api/v1/websites/:id/precheck/https
- select模式：检查3类风险
- acme模式：绕过校验
- critical风险：ok=false（阻止启用）
- warning风险：ok=true（允许启用但提示）

### 5. 风险解决API

- POST /api/v1/risks/:id/resolve
- 更新status为resolved
- 设置resolved_at
- 不做任何自动修复

## 技术亮点

1. **幂等生成机制**: 同一风险不会重复生成，依赖UNIQUE KEY约束
2. **Wildcard域名匹配**: 复用T2-07中实现的匹配规则
3. **弱覆盖判定**: 4个条件判断，推荐使用包含apex的证书
4. **前置预检**: 在用户启用HTTPS前检查风险，提前告知
5. **风险分级**: critical/warning/info，不同级别不同处理

## 部署说明

### 1. 数据库迁移

应用启动时自动创建certificate_risks表（MIGRATE=1）。

### 2. 环境变量

```bash
export RISK_SCANNER_ENABLED=1
export RISK_SCANNER_INTERVAL_SEC=300
export CERT_EXPIRING_DAYS=15
export CERT_EXPIRING_WEBSITE_THRESHOLD=2
export ACME_MAX_ATTEMPTS=3
```

### 3. 编译和启动

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
./bin/cmdb
```

### 4. 验证

```bash
./scripts/test_certificate_risks.sh
```

## API示例

### 前置预检（有风险）

**请求**:
```bash
curl -X POST "http://localhost:8080/api/v1/websites/1/precheck/https" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"cert_mode":"select","certificate_id":1}'
```

**响应**:
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
      }
    ]
  }
}
```

### 查询网站风险

**请求**:
```bash
curl -X GET "http://localhost:8080/api/v1/websites/1/risks" \
  -H "Authorization: Bearer <token>"
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

## 已知限制

1. 性能问题: 大量风险时扫描性能可能下降
2. 通知机制: 未实现风险通知
3. 风险历史: 未单独记录风险历史
4. 自动修复: 只发现和解释，不自动修复
5. 并发安全: Scanner是单线程的

## 后续改进

短期:
- 实现风险通知机制
- 优化扫描性能
- 添加风险统计面板

中期:
- 实现风险历史记录
- 添加风险趋势图
- 支持自定义风险规则

长期:
- 实现自动修复建议
- 添加风险预测
- 集成第三方监控平台

## 回滚策略

代码回滚:
```bash
git revert 8f8184797df9209b3efa3518cef6c339c2270bf8
```

数据库回滚:
```sql
DROP TABLE IF EXISTS certificate_risks;
```

禁用扫描:
```bash
export RISK_SCANNER_ENABLED=0
```

## 相关文档

- 完整交付报告: docs/T2-08-DELIVERY.md
- 实现计划: docs/T2-08-PLAN.md
- 测试脚本: scripts/test_certificate_risks.sh
- T2-07交付报告: docs/T2-07-DELIVERY.md
- T2-06交付报告: docs/T2-06-DELIVERY.md
- T2-05交付报告: docs/T2-05-DELIVERY.md

## 最终Commit

- Commit Hash: 8f8184797df9209b3efa3518cef6c339c2270bf8
- Commit Message: feat(T2-08): implement certificate risk detection and precheck system
- 仓库: labubu-daydayone/go_cmdb_web
- 分支: main
