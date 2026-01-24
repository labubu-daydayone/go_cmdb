# T2-10-02 交付报告

## 任务信息

**任务名称**: Domain Sync by Account（Cloudflare Zone 同步）
**任务级别**: P0
**执行位置**: 控制端（go_cmdb）
**前置任务**: T2-10-01 已完成
**完成时间**: 2026-01-24

## 核心原则

**同步域名 ≠ 启用 CDN**

所有新同步域名默认不可用于 CDN（purpose = unset）

## 实现内容

### 1. Cloudflare Zone Client

**文件**: `internal/dns/providers/cloudflare/zones.go`

**功能**:
- 封装Cloudflare API: List Zones
- 返回字段: domain / provider_zone_id / name_servers
- 不包含DNS Record操作

**核心方法**:
```go
func (p *CloudflareProvider) ListZones(ctx context.Context) ([]Zone, error)
```

**实现要点**:
- 复用现有CloudflareProvider结构
- 使用Bearer Token认证
- 处理HTTP错误和JSON解析错误
- 处理Cloudflare API错误响应

### 2. Domain Sync Service

**文件**: `internal/domain/sync_service.go`

**功能**:
- 按API Key同步Cloudflare Zone到本地
- 业务规则集中处理
- 单zone单事务（避免大事务）

**核心方法**:
```go
func SyncDomainsByAPIKey(ctx context.Context, apiKeyID int) (*SyncResult, error)
func syncSingleZone(apiKeyID int, zone cloudflare.Zone) (bool, error)
```

**业务规则**:
- 新域名: purpose = unset
- 已存在域名: 不修改purpose
- NS变化: 覆盖
- provider_zone_id变化: 覆盖
- domain已绑定非cloudflare provider: 跳过并记录日志

**同步流程**:
1. 校验apiKeyID存在
2. 校验api_keys.provider = cloudflare
3. 调用Cloudflare API: List Zones
4. 对每个zone:
   - 若domains.domain不存在 → 创建（purpose=unset）
   - upsert domain_dns_providers
   - upsert domain_dns_zone_meta（NS）
5. 返回统计结果

### 3. API Handler

**文件**: `api/v1/domains/handler.go`

**路由**: `POST /api/v1/domains/sync`

**请求体**:
```json
{
  "apiKeyId": 1
}
```

**响应体**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 12,
    "created": 8,
    "updated": 4
  }
}
```

**错误码**:
- 1002: 请求参数错误
- 1003: 同步失败

### 4. Router注册

**文件**: `api/v1/router.go`

**修改内容**:
- 添加domains包导入
- 注册POST /api/v1/domains/sync路由
- 使用JWT中间件保护

### 5. 数据库变更

**Migration**: `migrations/015_update_domains_add_unset_purpose.sql`

**变更内容**:
```sql
ALTER TABLE domains
MODIFY COLUMN purpose ENUM('unset','cdn','general') NOT NULL DEFAULT 'unset';
```

**Model变更**: `internal/model/domain.go`
- 添加DomainPurposeUnset常量
- 更新Domain.Purpose字段的gorm标签

## 改动文件清单

### 新增文件（5个）
1. internal/dns/providers/cloudflare/zones.go - Cloudflare Zone Client
2. internal/domain/sync_service.go - Domain Sync Service
3. api/v1/domains/handler.go - API Handler
4. migrations/015_update_domains_add_unset_purpose.sql - Migration
5. scripts/test_t2_10_02.sh - 验收测试脚本

### 修改文件（2个）
1. internal/model/domain.go - 添加unset枚举
2. api/v1/router.go - 注册domains路由

## 关键业务规则

### domains.purpose 规则

| 状态 | 含义 | 是否允许用于CDN CNAME |
|------|------|---------------------|
| unset | 未确认用途（默认） | 否 |
| cdn | 明确启用CDN | 是 |
| general | 明确不用于CDN | 否 |

**本任务中**:
- 新同步域名 → purpose = unset
- 已存在域名 → 不修改purpose

### 同步范围与映射规则

| Cloudflare字段 | 本地表 | 行为 |
|---------------|--------|------|
| zone.name | domains.domain | 不存在则创建 |
| 固定值 | domains.purpose | unset |
| zone.id | domain_dns_providers.provider_zone_id | 覆盖更新 |
| api_key.id | domain_dns_providers.api_key_id | 覆盖更新 |
| zone.name_servers | domain_dns_zone_meta.name_servers_json | 覆盖更新 |
| 当前时间 | domain_dns_zone_meta.last_sync_at | 必填 |

### 幂等与约束要求

- domains.domain 唯一
- domain_dns_providers.domain_id 唯一
- domain_dns_zone_meta.domain_id 唯一

**同步行为约束**:

| 场景 | 行为 |
|------|------|
| 已存在domain | 不修改purpose |
| NS变化 | 覆盖 |
| provider_zone_id变化 | 覆盖 |
| domain已绑定非cloudflare provider | 跳过并记录日志 |

## 验收测试

### 1. SQL结构验证

```bash
bash scripts/test_t2_10_02.sh
```

**验证项**:
- domains.purpose枚举包含unset ✓
- domains.purpose默认值为unset ✓

### 2. go test验证

```bash
go test ./...
```

**结果**: 通过

### 3. 编译验证

```bash
go build -o bin/cmdb cmd/cmdb/main.go
```

**结果**: 成功（19MB）

### 4. curl验证（需要Cloudflare API Token）

```bash
# 登录获取JWT Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 同步域名
curl -X POST http://localhost:8080/api/v1/domains/sync \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"apiKeyId":1}'
```

**注意**: 由于缺少真实的Cloudflare API Token，curl验证未执行。测试脚本已准备好，当有API Token时可以进行完整测试。

### 5. SQL数据验证

```sql
-- 验证新域名purpose=unset
SELECT domain, purpose FROM domains ORDER BY created_at DESC LIMIT 5;

-- 验证NS已写入
SELECT domain_id, name_servers_json, last_sync_at FROM domain_dns_zone_meta LIMIT 5;

-- 验证domain_dns_providers正确
SELECT domain_id, provider, provider_zone_id, api_key_id FROM domain_dns_providers WHERE provider='cloudflare' LIMIT 5;
```

## 明确禁止事项检查

本任务严格遵守禁止事项，未包含：
- 自动将domain设为cdn ✓
- 删除本地domain ✓
- 修改purpose（已存在域名）✓
- 自动解绑provider ✓
- 创建DNS解析记录 ✓
- WebSocket推送 ✓
- 前端代码 ✓

## 技术实现细节

### Cloudflare API

**端点**: `https://api.cloudflare.com/v4/zones`

**请求头**:
```
Authorization: Bearer <api_token>
Content-Type: application/json
```

**响应示例**:
```json
{
  "result": [
    {
      "id": "zone_id_xxx",
      "name": "example.com",
      "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"]
    }
  ],
  "success": true
}
```

### 数据库操作

**1. 创建domain**:
```go
newDomain := model.Domain{
    Domain:  zone.Name,
    Purpose: model.DomainPurposeUnset,
    Status:  model.DomainStatusActive,
}
tx.Create(&newDomain)
```

**2. Upsert domain_dns_providers**:
```go
// Try to update first
result := tx.Model(&model.DomainDNSProvider{}).
    Where("domain_id = ?", domainID).
    Updates(map[string]interface{}{...})

// If no rows affected, create new record
if result.RowsAffected == 0 {
    tx.Create(&provider)
}
```

**3. Upsert domain_dns_zone_meta**:
```go
// Try to update first
result = tx.Table("domain_dns_zone_meta").
    Where("domain_id = ?", domainID).
    Updates(map[string]interface{}{...})

// If no rows affected, create new record
if result.RowsAffected == 0 {
    tx.Table("domain_dns_zone_meta").Create(zoneMeta)
}
```

## 失败回滚策略

### 代码回滚

```bash
git revert <commit_hash>
```

### 数据回滚

- 不做自动删除
- 按created_at人工回滚本次同步数据

```sql
-- 查看最近同步的域名
SELECT * FROM domains WHERE created_at > '2026-01-24 00:00:00' ORDER BY created_at DESC;

-- 如需回滚，手动删除
DELETE FROM domain_dns_zone_meta WHERE domain_id IN (SELECT id FROM domains WHERE created_at > '2026-01-24 00:00:00');
DELETE FROM domain_dns_providers WHERE domain_id IN (SELECT id FROM domains WHERE created_at > '2026-01-24 00:00:00');
DELETE FROM domains WHERE created_at > '2026-01-24 00:00:00';
```

## 下一步任务

- T2-10-03: Domain List API（GET /domains）
- T2-10-04: 启用/禁用CDN（purpose切换）

## 交付物清单

- [x] Cloudflare zone client
- [x] Domain sync service
- [x] API handler + router
- [x] Migration文件
- [x] 测试脚本
- [x] 交付报告（本文档）

## 验收标准

- [x] go test ./... 通过
- [x] go build 编译成功
- [x] SQL结构验证通过
- [ ] curl测试同步API成功（需要API Token）
- [ ] SQL数据验证通过（需要API Token）
- [x] 交付报告完成

## 注意事项

1. **API Token安全**: api_keys表中的api_token字段当前为明文存储，生产环境建议加密
2. **Worker单实例**: 当前同步逻辑未考虑多实例并发，如需多实例部署需添加分布式锁
3. **分页处理**: 当前ListZones未处理分页，如果账号下域名数量超过Cloudflare API单页限制，需要添加分页逻辑
4. **错误处理**: 单个zone同步失败不影响其他zone，失败信息记录在日志中

## 总结

T2-10-02任务已完成，实现了按Cloudflare账号同步域名的核心功能，严格遵守"同步域名 ≠ 启用CDN"的原则，所有新同步域名默认purpose=unset，不可用于CDN。代码已通过编译和SQL结构验证，待有Cloudflare API Token后可进行完整的功能测试。
