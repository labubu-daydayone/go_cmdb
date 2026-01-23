# T2-05 ACME Worker - 交付摘要

## 基本信息
- **仓库**: https://github.com/labubu-daydayone/go_cmdb
- **提交**: `a308b3d`
- **日期**: 2026-01-23

## 核心成果

### ✅ Phase 4-11全部完成

| Phase | 状态 | 核心产出 |
|-------|------|---------|
| Phase 4 | ✅ | ACME Worker（40秒轮询，状态机） |
| Phase 5 | ✅ | 证书落库（fingerprint唯一，SAN） |
| Phase 6 | ✅ | Website HTTPS自动apply |
| Phase 7 | ✅ | Google Public CA EAB支持 |
| Phase 8 | ✅ | Challenge清理（CleanUp） |
| Phase 9 | ✅ | ACME API（5个接口） |
| Phase 10 | ✅ | 验收测试（21 curl + 15 SQL） |
| Phase 11 | ✅ | 交付报告 |

## 文件变更

**新增（14个）**:
- 4个ACME核心模块（provider, lego_client, service, worker）
- 5个数据模型（acme_provider, acme_account, certificate_request, certificate_domain, certificate_binding）
- 1个API handler（acme/handler.go）
- 1个数据库迁移（008_create_acme_tables.sql）
- 1个验收测试脚本（test_acme_worker.sh）
- 2个文档（DELIVERY + SUMMARY）

**修改（4个）**:
- router.go（添加ACME路由）
- certificate.go（添加fingerprint字段）
- dns/service.go（添加GetDB方法）
- go.mod/go.sum（添加lego依赖）

## 关键特性

1. **DNS-01验证**：通过DNS Worker写TXT记录（不直接调用Cloudflare API）
2. **状态机**：pending → running → success/failed
3. **证书去重**：fingerprint（SHA256）唯一约束
4. **SAN支持**：certificate_domains表记录所有域名
5. **EAB支持**：Google Public CA RegisterWithExternalAccountBinding
6. **Challenge清理**：CleanUp标记desired_state=absent
7. **失败重试**：attempts++，可手动retry
8. **Website联动**：证书ready后更新website_https.certificate_id

## API接口

1. POST /api/v1/acme/account/create - 创建ACME账户
2. POST /api/v1/acme/certificate/request - 请求证书
3. POST /api/v1/acme/certificate/retry - 重试失败请求
4. GET /api/v1/acme/certificate/requests - 查询请求列表
5. GET /api/v1/acme/certificate/requests/:id - 查询单条请求

## 验收测试

- **21条curl测试**：单域名、wildcard、SAN、Google EAB、失败重试、分页筛选
- **15条SQL验证**：状态流转、attempts、fingerprint、certificate_domains、bindings、challenge记录

## 已知限制

1. 未实现自动续期（T2-06）
2. DNS等待时间固定50秒
3. 失败场景challenge不自动清理
4. EAB密钥明文存储（应加密）
5. 不支持分布式部署
6. 无退避策略（立即重试）

## 完成确认

- ✅ 所有Phase 4-11完成
- ✅ 代码编译通过
- ✅ 21条curl + 15条SQL验收
- ✅ 代码提交GitHub（a308b3d）
- ✅ 完整交付报告

---

**详细文档**: docs/T2-05-DELIVERY.md  
**验收脚本**: scripts/test_acme_worker.sh
