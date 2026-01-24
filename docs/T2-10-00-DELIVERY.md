# T2-10-00 交付报告

任务名称：API Keys（Cloudflare）管理接口补齐
优先级：P0
执行位置：控制端（go_cmdb）
交付时间：2026-01-24

## 一、任务目标

让前端能通过接口完成 Cloudflare 账号密钥的新增/查询/启用停用/删除，并被后续 domains sync / dns worker 复用。

## 二、实现内容

### 1. API Keys CRUD + 状态切换

实现了以下5个接口：

#### 1.1 GET /api/v1/api-keys
查询列表（分页+搜索）

**查询参数**:
- page（默认 1）
- pageSize（默认 20，最大 100）
- keyword（匹配 name/account）
- provider（默认 cloudflare）
- status（active/inactive/all）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Cloudflare Account 1",
        "provider": "cloudflare",
        "account": "user@example.com",
        "apiTokenMasked": "****abcd",
        "status": "active",
        "createdAt": "2026-01-24T10:00:00+08:00",
        "updatedAt": "2026-01-24T10:00:00+08:00"
      }
    ],
    "total": 10,
    "page": 1,
    "pageSize": 20
  }
}
```

#### 1.2 POST /api/v1/api-keys/create
新增密钥（provider 先只允许 cloudflare）

**请求体**:
```json
{
  "name": "string",
  "provider": "cloudflare",
  "account": "string",
  "apiToken": "string"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

#### 1.3 POST /api/v1/api-keys/update
更新 name/account/apiToken/status（允许部分字段更新）

**请求体**:
```json
{
  "id": 1,
  "name": "string",
  "account": "string",
  "apiToken": "string",
  "status": "active"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

#### 1.4 POST /api/v1/api-keys/delete
删除（硬删，必须做依赖检查）

**请求体**:
```json
{
  "ids": [1, 2, 3]
}
```

**响应示例（成功）**:
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

**响应示例（失败）**:
```json
{
  "code": 3003,
  "message": "API Key 1 is being used by 5 domains, cannot delete",
  "data": null
}
```

#### 1.5 POST /api/v1/api-keys/toggle-status
启用/停用（便于前端开关）

**请求体**:
```json
{
  "id": 1,
  "status": "inactive"
}
```

**响应示例（成功）**:
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

**响应示例（失败）**:
```json
{
  "code": 3003,
  "message": "API Key is being used by 5 domains, cannot disable",
  "data": null
}
```

### 2. 响应字段规范（安全要求）

列表/详情响应中：
- apiToken 不允许明文返回
- 返回 apiTokenMasked（例如 ****abcd，只显示末 4 位）
- create/update 成功后不返回 token（避免日志泄露）

### 3. 依赖检查规则

删除/停用前必须检查 domain_dns_providers 表：
- 若有 domain 引用该 api_key_id → 返回 code=3003 + 提示信息
- 若无引用 → 允许操作

## 三、改动文件清单

### 新增文件（4个）

1. **api/v1/api_keys/handler.go**
   - ListAPIKeys：列表查询
   - CreateAPIKey：创建
   - UpdateAPIKey：更新
   - DeleteAPIKeys：删除
   - ToggleAPIKeyStatus：切换状态

2. **internal/api_keys/service.go**
   - Service层实现
   - CRUD操作
   - 依赖检查逻辑

3. **docs/T2-10-00-PLAN.md**
   - 实现计划

4. **scripts/test_t2_10_00.sh**
   - 验收测试脚本

### 修改文件（2个）

1. **api/v1/router.go**
   - 添加 api_keys import
   - 注册 /api/v1/api-keys/... 路由

2. **internal/model/api_key.go**
   - 添加 MaskedToken() 方法

## 四、数据库与迁移

### api_keys 表已存在且字段一致

表结构：
```sql
CREATE TABLE `api_keys` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `provider` enum('cloudflare') NOT NULL,
  `account` varchar(255) DEFAULT NULL,
  `api_token` varchar(255) NOT NULL,
  `status` enum('active','inactive') NOT NULL DEFAULT 'active',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

**无需新增迁移**。

## 五、验收测试结果

### go test 结果

```
go test ./... 通过
```

所有测试包编译通过，无错误。

### SQL 验证（4条）

1. **[SQL-1] 验证api_keys表结构** ✓
   - 验证表结构正确

2. **[SQL-2] 验证api_token字段存在但不在查询中** ✓
   - api_token字段存在

3. **[SQL-3] 插入测试数据并创建引用** ✓
   - 测试数据和引用已创建
   - 引用计数：1

4. **[SQL-4] 验证引用计数** ✓
   - 引用计数正确：1

### curl 验证（8条）

1. **[curl-0] 登录获取JWT Token** ✓
   - Token获取成功

2. **[curl-1] 创建API Key** ✓
   - 创建成功
   - 响应：`{"code":0,"data":null,"message":"success"}`

3. **[curl-2] 查询API Keys列表** ✓
   - 列表查询成功
   - Token已正确masked（****efgh）
   - 响应中不包含明文apiToken

4. **[curl-3] 更新API Key的name和account** ✓
   - 更新成功
   - 响应：`{"code":0,"data":null,"message":"success"}`

5. **[curl-4] 更新API Key的token** ✓
   - Token更新成功
   - 更新后Token仍然是masked

6. **[curl-5] 尝试禁用被引用的API Key（应该失败）** ✓
   - 依赖检查生效
   - 响应：`{"code":3003,"data":null,"message":"API Key is being used by 1 domains, cannot disable"}`

7. **[curl-6] 禁用未被引用的API Key（应该成功）** ✓
   - 禁用成功
   - 响应：`{"code":0,"data":null,"message":"success"}`

8. **[curl-7] 尝试删除被引用的API Key（应该失败）** ✓
   - 依赖检查生效
   - 响应：`{"code":3003,"data":null,"message":"API Key 9004 is being used by 1 domains, cannot delete"}`

9. **[curl-8] 删除未被引用的API Key（应该成功）** ✓
   - 删除成功
   - 响应：`{"code":0,"data":null,"message":"success"}`

### 验收总结

- **SQL验证**: 4/4 通过 ✓
- **curl验证**: 8/8 通过 ✓
- **go test**: 通过 ✓

## 六、失败回滚策略

### 代码回滚

```bash
git revert b28c2da
```

### 数据回滚

本任务仅新增接口，不动 schema，无需回滚表。

## 七、禁止事项检查

本任务严格遵守禁止事项：

- ✓ 禁止出现任何 yourapp/... import
- ✓ 禁止把路由写到 /api/v2
- ✓ 禁止返回明文 apiToken
- ✓ 禁止跳过引用检查直接删除

## 八、使用示例

### 1. 登录获取JWT Token

```bash
curl -X POST http://20.2.140.226:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expireAt": "2026-01-25T10:00:00+08:00",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 2. 创建API Key

```bash
TOKEN="your_jwt_token_here"

curl -X POST http://20.2.140.226:8080/api/v1/api-keys/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Cloudflare Account 1",
    "provider": "cloudflare",
    "account": "user@example.com",
    "apiToken": "your_cloudflare_api_token_here"
  }'
```

### 3. 查询API Keys列表

```bash
curl -X GET "http://20.2.140.226:8080/api/v1/api-keys?page=1&pageSize=20&status=all" \
  -H "Authorization: Bearer $TOKEN"
```

### 4. 更新API Key

```bash
curl -X POST http://20.2.140.226:8080/api/v1/api-keys/update \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "name": "Cloudflare Account 1 Updated",
    "account": "newuser@example.com"
  }'
```

### 5. 切换状态

```bash
curl -X POST http://20.2.140.226:8080/api/v1/api-keys/toggle-status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "status": "inactive"
  }'
```

### 6. 删除API Key

```bash
curl -X POST http://20.2.140.226:8080/api/v1/api-keys/delete \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [1, 2, 3]
  }'
```

## 九、注意事项

### 1. 安全性

- API Token 明文存储在数据库中（生产环境建议加密）
- 响应中不返回明文 Token（只返回 masked）
- 所有接口都走 JWT 认证

### 2. 依赖检查

- 删除/停用前必须检查 domain_dns_providers 引用
- 被引用时返回 code=3003 并提示引用数量

### 3. 分页限制

- 默认 pageSize=20
- 最大 pageSize=100

### 4. 错误码

- 0: 成功
- 1001: 未授权（缺少 JWT Token）
- 3000: 参数错误
- 3003: 依赖检查失败（被引用）

## 十、后续优化建议

1. **API Token 加密存储**
   - 当前为明文存储
   - 建议使用 AES 加密

2. **批量操作优化**
   - 当前 delete 支持批量，但每个 ID 单独检查
   - 可优化为批量检查

3. **审计日志**
   - 记录 API Key 的创建/更新/删除操作
   - 便于安全审计

4. **Provider 扩展**
   - 当前只支持 cloudflare
   - 后续可扩展 aliyun/tencent/huawei

## 十一、交付清单

- [x] 实现 5 个 API 接口
- [x] 实现依赖检查逻辑
- [x] 实现 Token Masked 返回
- [x] go test 通过
- [x] 4 条 SQL 验证通过
- [x] 8 条 curl 验证通过
- [x] 编写交付报告
- [x] 代码提交到 GitHub

## 十二、Git 提交信息

**Commit**: b28c2da
**Message**: feat(T2-10-00): add API Keys management interfaces (CRUD + status toggle + dependency check)
**Repository**: https://github.com/labubu-daydayone/go_cmdb

---

交付完成时间：2026-01-24
交付人：AI Assistant
