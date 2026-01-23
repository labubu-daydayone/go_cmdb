# T2-07 实现计划：证书与网站关系可视化 + HTTPS启用前风险校验

## 任务概述

实现证书与网站关系的风险治理层，解决三个生产问题：
1. 一个证书被哪些网站使用？
2. 一个网站启用HTTPS时，证书是否真正覆盖它的所有域名？
3. 在错误证书绑定前，把风险挡在控制端

## 核心设计原则

1. 不修改ACME/DNS/Agent行为
2. 所有校验发生在控制端
3. 校验失败：不落库、不apply_config、必须给出明确失败原因
4. 校验是前置条件，不是事后检查

## 实现阶段（8个Phase）

### Phase 1: 任务分析（当前）

输出：
- 实现计划文档（本文档）
- 技术方案设计
- 文件结构规划

### Phase 2: 证书域名覆盖判定逻辑

核心功能：
- 实现wildcard域名匹配规则
- 实现覆盖状态判定（covered/partial/not_covered）
- 编写单元测试

文件：
- internal/cert/coverage.go
- internal/cert/coverage_test.go

关键逻辑：
```
wildcard *.example.com 覆盖：
- a.example.com ✓
- b.example.com ✓

不覆盖：
- example.com ✗
- a.b.example.com ✗
```

覆盖状态：
- covered: certificate_domains完全覆盖website_domains
- partial: 至少覆盖一个，但不是全部
- not_covered: 一个都不覆盖

### Phase 3: 证书→网站关系查询API

API: GET /api/v1/certificates/{id}/websites

返回内容：
- certificate_id
- domains（来自certificate_domains）
- websites列表：
  - website_id
  - website_primary_domain
  - website_domains（全量）
  - https_enabled
  - bind_status（active/inactive）

实现：
- 通过certificate_bindings表反查
- 一个证书可绑定多个网站
- inactive的binding也要能查出来（用于审计）

文件：
- internal/cert/service.go
- api/v1/cert/handler.go

### Phase 4: 网站→可用证书列表API

API: GET /api/v1/websites/{id}/certificates/candidates

返回内容：
- website_id
- website_domains
- candidates列表：
  - certificate_id
  - certificate_domains
  - coverage_status（covered/partial/not_covered）
  - missing_domains（若partial）
  - expire_at
  - provider

实现：
- 查询所有证书
- 对每个证书调用coverage判定函数
- 返回覆盖状态和缺失域名

文件：
- internal/cert/service.go（扩展）
- api/v1/cert/handler.go（扩展）

### Phase 5: HTTPS启用前强校验

API: POST /api/v1/websites/https/enable（修改已有接口）

校验逻辑：

当cert_mode=select且certificate_id指定时：
- 校验该certificate是否存在
- 校验certificate_domains是否100%覆盖website_domains
- 只要有一个domain未覆盖：
  - 拒绝请求
  - 返回业务错误码3003
  - message明确指出哪些域名未覆盖

当cert_mode=acme：
- 不做覆盖校验（因为ACME由系统生成）
- 但必须保证：
  - website_domains非空
  - domain合法（简单校验即可）

验收点：
- partial覆盖的证书不能被选中
- 前端在"选择证书"时即可用candidates API做提示
- 后端是最终防线，不能被绕过

文件：
- api/v1/websites/handler.go（修改已有）

### Phase 6: 错误信息优化

所有校验失败必须返回：
- code: 非0
- message: 人类可读
- data（可选）：
  - missing_domains
  - certificate_domains
  - website_domains

禁止只返回一句"invalid certificate"

示例：
```json
{
  "code": 3003,
  "message": "Certificate does not cover all website domains",
  "data": {
    "certificate_domains": ["*.example.com"],
    "website_domains": ["example.com", "www.example.com"],
    "missing_domains": ["example.com"]
  }
}
```

### Phase 7: 验收测试

CURL测试（15+条）：
1. 查询certificate → websites
2. 查询website → certificate candidates
3. partial覆盖拒绝HTTPS enable
4. covered覆盖成功enable
5. acme模式绕过校验
6. 一个证书绑定多个网站
7. wildcard匹配测试（多种场景）
8. 错误返回格式验证

SQL验证（10+条）：
1. certificate_bindings关系
2. website_domains与certificate_domains对照
3. 校验失败时无副作用写入
4. 覆盖状态统计
5. 多网站绑定验证

文件：
- scripts/test_certificate_coverage.sh

### Phase 8: 交付报告

内容：
1. 基本信息（仓库/提交/日期）
2. 新增/修改文件清单
3. 覆盖判定规则说明（文字+示例）
4. curl（>=15）
5. SQL（>=10）
6. 失败示例与错误返回
7. 回滚说明
8. 已知限制

文件：
- docs/T2-07-DELIVERY.md
- docs/T2-07-SUMMARY.md

## 必须覆盖的业务场景

### 场景1: partial覆盖（缺apex域名）
- 网站域名: example.com, www.example.com
- 证书域名: *.example.com
- 结果: partial（缺example.com）

### 场景2: covered覆盖
- 网站域名: a.example.com
- 证书域名: *.example.com
- 结果: covered

### 场景3: not_covered（二级wildcard不覆盖）
- 网站域名: a.b.example.com
- 证书域名: *.example.com
- 结果: not_covered

### 场景4: 一个证书绑定多个网站
- 查询反向关系正确

### 场景5: HTTPS启用失败时
- 不写website_https
- 不写config_versions
- 不下发agent_tasks

## 技术约束

1. 不允许新增大表
2. 只允许：
   - 新增model文件（如果缺）
   - 新增service层
   - 新增handler

3. 推荐新增文件：
   - internal/cert/coverage.go
   - internal/cert/service.go
   - api/v1/cert/handler.go
   - api/v1/website/https_handler.go（或改已有）

## 回滚策略

- git revert本任务提交
- 回滚后系统行为恢复为：
  - HTTPS enable不做覆盖校验（旧行为）
- 不影响已存在证书与网站关系

## 关键技术点

### 1. Wildcard域名匹配

```go
// *.example.com 匹配规则
func matchWildcard(wildcardDomain, targetDomain string) bool {
    // 去掉 *. 前缀
    baseDomain := strings.TrimPrefix(wildcardDomain, "*.")
    
    // 目标域名必须以 .baseDomain 结尾
    if !strings.HasSuffix(targetDomain, "."+baseDomain) {
        return false
    }
    
    // 去掉后缀后，剩余部分不能包含点（防止二级wildcard）
    prefix := strings.TrimSuffix(targetDomain, "."+baseDomain)
    return !strings.Contains(prefix, ".")
}
```

### 2. 覆盖状态判定

```go
func CalculateCoverage(certDomains, websiteDomains []string) CoverageResult {
    covered := []string{}
    missing := []string{}
    
    for _, wd := range websiteDomains {
        if isCoveredBy(wd, certDomains) {
            covered = append(covered, wd)
        } else {
            missing = append(missing, wd)
        }
    }
    
    if len(missing) == 0 {
        return CoverageResult{Status: "covered"}
    } else if len(covered) > 0 {
        return CoverageResult{Status: "partial", Missing: missing}
    } else {
        return CoverageResult{Status: "not_covered", Missing: missing}
    }
}
```

### 3. HTTPS启用校验

```go
func (h *WebsiteHandler) EnableHTTPS(c *gin.Context) {
    // ... 解析请求 ...
    
    if req.CertMode == "select" && req.CertificateID > 0 {
        // 获取证书域名
        certDomains := getCertificateDomains(req.CertificateID)
        
        // 获取网站域名
        websiteDomains := getWebsiteDomains(req.WebsiteID)
        
        // 计算覆盖状态
        coverage := CalculateCoverage(certDomains, websiteDomains)
        
        // 只有完全覆盖才允许
        if coverage.Status != "covered" {
            httpx.FailErr(c, httpx.ErrStateConflict(
                "Certificate does not cover all website domains",
            ).WithData(gin.H{
                "certificate_domains": certDomains,
                "website_domains": websiteDomains,
                "missing_domains": coverage.Missing,
            }))
            return
        }
    }
    
    // 继续原有逻辑...
}
```

## 数据流图

```
1. 证书→网站关系查询
   certificate_id → certificate_bindings → website_id → website_domains

2. 网站→可用证书列表
   website_id → website_domains → coverage判定 → candidates列表

3. HTTPS启用校验
   website_id + certificate_id → coverage判定 → 允许/拒绝
```

## 预期输出

### 新增文件（4-5个）
1. internal/cert/coverage.go
2. internal/cert/coverage_test.go
3. internal/cert/service.go
4. api/v1/cert/handler.go
5. scripts/test_certificate_coverage.sh

### 修改文件（2-3个）
1. api/v1/websites/handler.go（添加校验逻辑）
2. api/v1/router.go（添加新路由）
3. internal/httpx/errors.go（可能需要扩展错误返回）

### 新增API（2个）
1. GET /api/v1/certificates/{id}/websites
2. GET /api/v1/websites/{id}/certificates/candidates

### 修改API（1个）
1. POST /api/v1/websites/https/enable（添加校验）

## 风险点

1. wildcard匹配规则复杂，需要充分测试
2. 覆盖判定逻辑需要考虑边界情况
3. HTTPS启用校验可能影响现有流程
4. 错误信息需要清晰易懂

## 时间估算

- Phase 1: 30分钟（任务分析）
- Phase 2: 1小时（coverage逻辑+单测）
- Phase 3: 45分钟（证书→网站API）
- Phase 4: 45分钟（网站→证书API）
- Phase 5: 1小时（HTTPS校验）
- Phase 6: 30分钟（错误信息优化）
- Phase 7: 1小时（验收测试）
- Phase 8: 45分钟（交付报告）

总计：约6小时

---

开始时间：2026-01-23
预计完成：2026-01-23
