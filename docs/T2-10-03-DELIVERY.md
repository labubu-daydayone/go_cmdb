# T2-10-03 交付报告

## 任务信息

**任务编号**: T2-10-03  
**任务名称**: Domain List API（GET /domains）  
**优先级**: P0  
**开发位置**: 控制端（go_cmdb）  
**提交时间**: 2026-01-24  
**提交人**: AI Assistant  

## 任务目标

实现域名列表查询接口，聚合展示 domains/provider/apiKey/NS 元数据，支持分页和过滤。

## 完成内容

### 1. Service层实现

**文件**: `internal/domain/list_service.go`

**核心方法**:
- `ListDomains`: 聚合查询域名列表

**聚合逻辑**:
```sql
SELECT 
  d.id, d.domain, d.purpose, d.status,
  p.provider,
  ak.id as api_key_id, ak.name as api_key_name,
  zm.name_servers_json, zm.last_sync_at,
  d.created_at
FROM domains d
LEFT JOIN domain_dns_providers p ON p.domain_id = d.id
LEFT JOIN api_keys ak ON ak.id = p.api_key_id
LEFT JOIN domain_dns_zone_meta zm ON zm.domain_id = d.id
```

**过滤支持**:
- `keyword`: 域名模糊搜索
- `purpose`: unset/cdn/general
- `provider`: cloudflare
- `apiKeyId`: 按账号筛选
- `status`: active/inactive

**分页**:
- 默认 page=1, pageSize=20
- 最大 pageSize=100

**数据处理**:
- `nameServers` 从 JSON 解析为数组
- `apiKey` 信息组装为对象
- 空值处理（provider/apiKey/nameServers/lastSyncAt 可为 null）

### 2. Handler层实现

**文件**: `api/v1/domains/handler.go`

**新增方法**: `ListDomains`

**查询参数**:
- `page` (默认 1)
- `pageSize` (默认 20, 最大 100)
- `keyword` (域名模糊搜索)
- `purpose` (unset/cdn/general)
- `provider` (cloudflare)
- `apiKeyId` (按账号筛选)
- `status` (active/inactive)

**响应格式**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "domain": "example.com",
        "purpose": "unset",
        "status": "active",
        "provider": "cloudflare",
        "apiKey": {
          "id": 1,
          "name": "Cloudflare Account 1"
        },
        "nameServers": [
          "ada.ns.cloudflare.com",
          "bob.ns.cloudflare.com"
        ],
        "lastSyncAt": "2026-01-24T10:00:00+08:00",
        "createdAt": "2026-01-24T09:00:00+08:00"
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

### 3. Router注册

**文件**: `api/v1/router.go`

**路由**: `GET /api/v1/domains`

**认证**: JWT middleware（必须登录）

## 改动文件清单

### 新增文件（2个）
1. `internal/domain/list_service.go` - 域名列表查询服务
2. `scripts/test_t2_10_03.sh` - 验收测试脚本

### 修改文件（2个）
1. `api/v1/domains/handler.go` - 添加 ListDomains 方法
2. `api/v1/router.go` - 添加 GET /api/v1/domains 路由

## 验收测试

### go test
```bash
$ go test ./...
ok  	go_cmdb/internal/auth	(cached)
ok  	go_cmdb/internal/cert	(cached)
ok  	go_cmdb/internal/config	(cached)
ok  	go_cmdb/internal/dns	(cached)
ok  	go_cmdb/internal/httpx	(cached)
```

**结果**: 通过 ✓

### SQL验证（3条）

**[SQL-1] 插入测试数据**
- 插入3个测试域名（purpose=unset/cdn/general）
- 为第一个域名绑定provider和NS
- 为第二个域名绑定provider（无NS）

**结果**: 通过 ✓

**[SQL-2] 验证purpose=unset的域名**
```sql
SELECT COUNT(*) FROM domains 
WHERE domain LIKE 'test-list-%' AND purpose='unset';
```
**期望**: 1  
**实际**: 1  
**结果**: 通过 ✓

**[SQL-3] 验证NS不为空（同步过的域名）**
```sql
SELECT COUNT(*) FROM domain_dns_zone_meta zm 
JOIN domains d ON zm.domain_id = d.id 
WHERE d.domain LIKE 'test-list-%' 
  AND zm.name_servers_json IS NOT NULL 
  AND zm.name_servers_json != '';
```
**期望**: 1  
**实际**: 1  
**结果**: 通过 ✓

### curl验证（5条）

**[curl-1] 基本查询（GET /api/v1/domains）**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?page=1&pageSize=20"
```
**期望**: code=0, 返回域名包含test-list-1.com  
**实际**: code=0, 返回域名包含test-list-1.com  
**结果**: 通过 ✓

**[curl-2] purpose过滤（purpose=unset）**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?purpose=unset"
```
**期望**: code=0, 返回域名包含test-list-1.com（purpose=unset）  
**实际**: code=0, 返回域名包含test-list-1.com（purpose=unset）  
**结果**: 通过 ✓

**[curl-3] purpose过滤（purpose=cdn）**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?purpose=cdn"
```
**期望**: code=0, 返回域名包含test-list-2.com（purpose=cdn）  
**实际**: code=0, 返回域名包含test-list-2.com（purpose=cdn）  
**结果**: 通过 ✓

**[curl-4] 验证NS字段**
**期望**: 返回的域名包含ada.ns.cloudflare.com  
**实际**: 返回的域名包含ada.ns.cloudflare.com  
**结果**: 通过 ✓

**[curl-5] 验证apiKey字段**
**期望**: 返回的域名包含test_list_key_1  
**实际**: 返回的域名包含test_list_key_1  
**结果**: 通过 ✓

### 验收总结

- **SQL验证**: 3/3 通过
- **curl验证**: 5/5 通过
- **go test**: 通过

## 核心设计原则

### 1. 只读接口
- 本接口仅用于查询展示，不涉及任何写操作
- 不修改 purpose、不删除 domain、不解绑 provider

### 2. 聚合展示
- 一次查询返回所有相关信息
- 避免前端多次请求（N+1问题）

### 3. purpose语义严格
- `unset`: 已同步但未决定用途
- `cdn`: 允许 CDN 使用
- `general`: 禁止 CDN 使用

### 4. 空值处理
- `provider`: 未绑定时为 null
- `apiKey`: 未绑定时为 null
- `nameServers`: 未同步时为空数组 []
- `lastSyncAt`: 未同步时为 null

## 禁止事项检查

本任务严格遵守禁止事项，未包含：
- ✓ 禁止修改 purpose
- ✓ 禁止删除 domain
- ✓ 禁止解绑 provider
- ✓ 禁止创建 DNS 解析记录
- ✓ 禁止 WebSocket 推送
- ✓ 禁止前端代码

## 使用示例

### 登录
```bash
curl -X POST http://20.2.140.226:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 基本查询
```bash
TOKEN="your_jwt_token_here"

curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?page=1&pageSize=20"
```

### purpose过滤
```bash
# 查询未决定用途的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?purpose=unset"

# 查询允许CDN使用的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?purpose=cdn"

# 查询禁止CDN使用的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?purpose=general"
```

### provider过滤
```bash
# 查询Cloudflare管理的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?provider=cloudflare"
```

### apiKeyId过滤
```bash
# 查询指定账号管理的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?apiKeyId=1"
```

### keyword搜索
```bash
# 模糊搜索域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?keyword=example"
```

### 组合过滤
```bash
# 查询Cloudflare管理的、允许CDN使用的域名
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/domains?provider=cloudflare&purpose=cdn&page=1&pageSize=50"
```

## 注意事项

1. **分页限制**: 默认20，最大100
2. **空值处理**: provider/apiKey/nameServers/lastSyncAt 可能为 null 或空数组
3. **NS数组**: 从 JSON 字段解析，失败时返回空数组
4. **性能**: 使用 LEFT JOIN 聚合查询，避免 N+1 问题
5. **过滤逻辑**: 多个过滤条件使用 AND 连接

## Git提交信息

**Commit**: 9ad94e1  
**Message**: feat(T2-10-03): add Domain List API (GET /domains)  
**仓库**: https://github.com/labubu-daydayone/go_cmdb  
**分支**: main  

## 交付清单

- [x] Service层实现（ListDomains）
- [x] Handler层实现（ListDomains）
- [x] Router注册（GET /api/v1/domains）
- [x] 验收测试脚本（test_t2_10_03.sh）
- [x] SQL验证（3/3 通过）
- [x] curl验证（5/5 通过）
- [x] go test（通过）
- [x] 禁止事项检查（通过）
- [x] 交付文档（本文档）

## 后续任务

- T2-10-04: 启用/禁用CDN（purpose切换）
- T2-10-05: 域名详情API（GET /domains/:id）

---

**交付完成时间**: 2026-01-24 18:12:00  
**测试环境**: 20.2.140.226:8080  
**服务状态**: 运行中  
