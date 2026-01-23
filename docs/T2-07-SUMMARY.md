# T2-07 证书与网站关系可视化 + HTTPS启用前风险校验 - 交付总结

## 核心交付

Commit: `cf40a5d5443a28efaaaf8753c0453af0ec6fbbd2`

实现三个生产问题的解决方案：
1. 一个证书被哪些网站使用？（证书→网站关系查询）
2. 一个网站启用HTTPS时，证书是否真正覆盖它的所有域名？（覆盖状态判定）
3. 在错误证书绑定前，把风险挡在控制端（HTTPS启用前强校验）

## 文件变更

新增6个文件：
- internal/cert/coverage.go（覆盖判定逻辑）
- internal/cert/coverage_test.go（24个单元测试）
- internal/cert/service.go（证书服务层）
- api/v1/cert/handler.go（证书API Handler）
- scripts/test_certificate_coverage.sh（验收测试）
- docs/T2-07-PLAN.md（实现计划）

修改4个文件：
- internal/httpx/errors.go（添加Data字段）
- internal/httpx/resp.go（支持Data返回）
- api/v1/websites/handler.go（添加覆盖校验）
- api/v1/router.go（注册新路由）

## API路由

新增2个API：
1. GET /api/v1/certificates/{id}/websites
   - 查询使用证书的所有网站
   - 返回网站列表（ID/域名/HTTPS状态/绑定状态）

2. GET /api/v1/websites/{id}/certificates/candidates
   - 查询网站的证书候选列表
   - 返回证书列表（ID/域名/覆盖状态/缺失域名）

修改2个API：
1. POST /api/v1/websites（Create）
   - 新增校验：select模式必须100%覆盖
   - 新增校验：acme模式域名非空

2. POST /api/v1/websites/{id}（Update）
   - 新增校验：select模式必须100%覆盖

## Wildcard匹配规则

*.example.com匹配：
- a.example.com（一级子域名）
- www.example.com（一级子域名）

*.example.com不匹配：
- example.com（apex域名）
- a.b.example.com（二级子域名）

## 覆盖状态

1. covered：完全覆盖（所有网站域名都被证书覆盖）
2. partial：部分覆盖（至少一个域名被覆盖，但不是全部）
3. not_covered：不覆盖（一个域名都不覆盖）

## 错误返回格式

partial覆盖示例（HTTP 409）：
```json
{
  "code": 3003,
  "message": "Certificate does not cover all website domains",
  "data": {
    "certificateDomains": ["*.example.com"],
    "websiteDomains": ["example.com", "www.example.com"],
    "missingDomains": ["example.com"],
    "coverageStatus": "partial"
  }
}
```

## 验收测试

测试脚本：scripts/test_certificate_coverage.sh

CURL测试（20条）：
- 登录获取token
- 查询证书→网站（有效/无效）
- 查询网站→证书候选（有效/无效）
- partial覆盖拒绝（3003错误）
- covered覆盖成功
- ACME模式绕过校验
- 无效证书ID（404错误）
- ACME模式空域名（2001错误）
- wildcard匹配测试
- 二级子域名测试
- 错误响应格式验证
- 未授权访问测试

SQL验证（15条）：
- 验证表存在（certificate_bindings/certificate_domains/website_domains）
- 统计证书/网站数量
- 验证wildcard域名
- 验证is_wildcard标志
- 验证website_https字段
- 统计HTTPS启用数量
- 统计cert_mode分布
- 验证无重复域名

总计35个测试用例。

## 快速验证

```bash
# 编译
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go

# 启动服务
./bin/cmdb

# 运行测试
./scripts/test_certificate_coverage.sh
```

## 回滚策略

```bash
# 代码回滚
cd /home/ubuntu/go_cmdb_new
git revert cf40a5d5443a28efaaaf8753c0453af0ec6fbbd2

# 或者禁用校验（注释掉validateCertificateCoverage调用）
```

## 已知限制

1. 查询性能：大量绑定时存在N+1查询问题
2. 无缓存机制：每次请求都重新计算覆盖状态
3. 无批量操作：只支持单个网站的HTTPS启用校验
4. 无国际化：错误信息全部为英文

## 后续改进

短期（1-2周）：
- 优化查询性能（JOIN + 索引）
- 添加Redis缓存
- 添加分页支持

中期（1-2个月）：
- 批量HTTPS启用API
- 证书覆盖状态监控面板
- 证书→网站关系图可视化

长期（3-6个月）：
- 证书覆盖状态预警
- 自定义覆盖规则
- 证书推荐功能

## 相关文档

- 完整交付报告：docs/T2-07-DELIVERY.md
- 实现计划：docs/T2-07-PLAN.md
- 测试脚本：scripts/test_certificate_coverage.sh
- T2-06交付报告：docs/T2-06-DELIVERY.md（证书自动续期）
- T2-05交付报告：docs/T2-05-DELIVERY.md（ACME Worker）

---

交付日期：2026-01-23
