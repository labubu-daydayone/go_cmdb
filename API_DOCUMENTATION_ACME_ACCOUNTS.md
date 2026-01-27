# ACME账号管理API接口文档

版本：v1.0  
更新时间：2026-01-27  
基础URL：http://20.2.140.226:8080

---

## 通用说明

### 请求规范

- 所有接口仅使用 **GET** 和 **POST** 方法
- Content-Type：application/json
- 字符编码：UTF-8

### 响应格式

所有接口统一返回以下格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "pageSize": 20
  }
}
```

### 错误码

| 错误码 | 说明 | 场景 |
|--------|------|------|
| 0 | 成功 | 所有成功操作 |
| 1001 | 参数错误 | 缺少必填参数、参数格式错误 |
| 2001 | 资源不存在 | 查询的资源不存在 |
| 2002 | 参数验证失败 | 参数不符合验证规则 |
| 3002 | 业务逻辑错误 | 违反业务约束（如删除默认账号） |
| 5000 | 服务器内部错误 | 数据库错误、系统异常 |

---

## 接口列表

### 1. 获取ACME Provider列表

获取所有可用的CA提供商列表。

**接口地址**：`GET /api/v1/acme/providers`

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "letsencrypt",
        "directoryUrl": "https://acme-v02.api.letsencrypt.org/directory",
        "requiresEab": false,
        "status": "active",
        "createdAt": "2026-01-20T10:00:00Z",
        "updatedAt": "2026-01-20T10:00:00Z"
      },
      {
        "id": 2,
        "name": "google",
        "directoryUrl": "https://dv.acme-v02.api.pki.goog/directory",
        "requiresEab": true,
        "status": "active",
        "createdAt": "2026-01-20T10:00:00Z",
        "updatedAt": "2026-01-20T10:00:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 2
  }
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | Provider ID |
| name | string | Provider名称（letsencrypt/google） |
| directoryUrl | string | ACME目录URL |
| requiresEab | boolean | 是否需要EAB凭证 |
| status | string | 状态（active/inactive） |
| createdAt | string | 创建时间（ISO 8601格式） |
| updatedAt | string | 更新时间（ISO 8601格式） |

**Provider说明**：

- **letsencrypt**：Let's Encrypt，免费CA，无需EAB凭证
- **google**：Google Trust Services，需要EAB凭证

---

### 2. 创建ACME账号

在指定的CA提供商创建新的ACME账号。

**接口地址**：`POST /api/v1/acme/account/create`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| providerName | string | 是 | Provider名称（letsencrypt/google） |
| email | string | 是 | 邮箱地址 |
| eabKid | string | 否 | EAB Key ID（Google需要） |
| eabHmacKey | string | 否 | EAB HMAC Key（Google需要） |

**请求示例**：

```json
{
  "providerName": "letsencrypt",
  "email": "admin@example.com",
  "eabKid": "",
  "eabHmacKey": ""
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "providerId": 1,
        "email": "admin@example.com",
        "status": "pending"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 账号ID |
| providerId | int | Provider ID |
| email | string | 邮箱地址 |
| status | string | 账号状态（pending/active/disabled） |
| eabKid | string | EAB Key ID（仅在提供时返回） |

**错误响应**：

```json
{
  "code": 2002,
  "message": "Key: 'CreateAccountRequest.ProviderName' Error:Field validation for 'ProviderName' failed on the 'required' tag",
  "data": null
}
```

**业务规则**：

1. 同一Provider下，同一邮箱只能创建一个账号
2. 如果Provider的requiresEab为true，必须提供eabKid和eabHmacKey
3. 账号创建后状态为pending，需要等待ACME服务器确认

---

### 3. 获取ACME账号列表

获取ACME账号列表，支持分页和过滤。

**接口地址**：`GET /api/v1/acme/accounts`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| pageSize | int | 否 | 每页数量，默认20 |
| providerId | int | 否 | 按Provider ID过滤 |
| status | string | 否 | 按状态过滤（active/disabled） |
| email | string | 否 | 按邮箱模糊搜索 |

**请求示例**：

```
GET /api/v1/acme/accounts?page=1&pageSize=20&providerId=1&status=active
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "providerId": 1,
        "providerName": "letsencrypt",
        "email": "admin@example.com",
        "status": "active",
        "isDefault": true,
        "accountUrl": "https://acme-v02.api.letsencrypt.org/acme/acct/123456",
        "createdAt": "2026-01-26T10:30:00Z",
        "updatedAt": "2026-01-26T10:30:00Z"
      },
      {
        "id": 13,
        "providerId": 1,
        "providerName": "letsencrypt",
        "email": "backup@example.com",
        "status": "disabled",
        "isDefault": false,
        "accountUrl": "https://acme-v02.api.letsencrypt.org/acme/acct/789012",
        "createdAt": "2026-01-25T08:15:00Z",
        "updatedAt": "2026-01-26T14:20:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 20
  }
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 账号ID |
| providerId | int | Provider ID |
| providerName | string | Provider名称 |
| email | string | 邮箱地址 |
| status | string | 账号状态（active/disabled） |
| isDefault | boolean | 是否为该Provider的默认账号 |
| accountUrl | string | ACME账号URL |
| createdAt | string | 创建时间（ISO 8601格式） |
| updatedAt | string | 更新时间（ISO 8601格式） |

---

### 4. 设置默认ACME账号

将指定账号设置为该Provider的默认账号。

**接口地址**：`POST /api/v1/acme/accounts/set-default`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| providerId | int | 是 | Provider ID |
| accountId | int | 是 | 账号ID |

**请求示例**：

```json
{
  "providerId": 1,
  "accountId": 12
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "providerId": 1,
        "accountId": 12
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**业务规则**：

1. 每个Provider只能有一个默认账号
2. 设置新默认账号会自动取消旧的默认状态
3. 只能设置status为active的账号为默认
4. 操作是原子性的（使用Upsert语义）

---

### 5. 启用ACME账号

启用已禁用的ACME账号。

**接口地址**：`POST /api/v1/acme/accounts/enable`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 账号ID |

**请求示例**：

```json
{
  "id": 13
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 13,
        "status": "active"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**业务规则**：

1. 只能启用status为disabled的账号
2. 启用后账号状态变为active
3. 启用操作不会自动设置为默认账号
4. 操作是幂等的（重复启用不会报错）

---

### 6. 禁用ACME账号

禁用ACME账号，禁用后不能用于证书申请。

**接口地址**：`POST /api/v1/acme/accounts/disable`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 账号ID |

**请求示例**：

```json
{
  "id": 13
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 13,
        "status": "disabled"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**错误响应（禁用默认账号）**：

```json
{
  "code": 3002,
  "message": "cannot disable default account, please set another account as default first",
  "data": null
}
```

**业务规则**：

1. 不能禁用默认账号（isDefault为true）
2. 如需禁用默认账号，需先设置其他账号为默认
3. 禁用后账号状态变为disabled
4. 操作是幂等的（重复禁用不会报错）

---

### 7. 删除ACME账号

删除ACME账号，删除后无法恢复。

**接口地址**：`POST /api/v1/acme/accounts/delete`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 账号ID |

**请求示例**：

```json
{
  "id": 13
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 13,
        "deleted": true
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**错误响应（删除默认账号）**：

```json
{
  "code": 3002,
  "message": "cannot delete default account, please set another account as default first",
  "data": null
}
```

**错误响应（账号被引用）**：

```json
{
  "code": 3002,
  "message": "cannot delete account, it is referenced by certificate requests",
  "data": null
}
```

或

```json
{
  "code": 3002,
  "message": "cannot delete account, it is referenced by certificates",
  "data": null
}
```

**业务规则**：

1. 不能删除默认账号（isDefault为true）
2. 不能删除被certificate_requests表引用的账号
3. 不能删除被certificates表引用的账号
4. 删除操作会物理删除数据库记录
5. 如需删除默认账号，需先设置其他账号为默认

---

## 使用流程

### 典型使用场景

#### 场景1：创建Let's Encrypt账号

```bash
# 1. 获取Provider列表
curl "http://20.2.140.226:8080/api/v1/acme/providers"

# 2. 创建账号（providerName: letsencrypt）
curl -X POST http://20.2.140.226:8080/api/v1/acme/account/create \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "letsencrypt",
    "email": "admin@example.com",
    "eabKid": "",
    "eabHmacKey": ""
  }'

# 3. 设置为默认账号
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/set-default \
  -H "Content-Type: application/json" \
  -d '{
    "providerId": 1,
    "accountId": 12
  }'
```

#### 场景2：创建Google Trust Services账号

```bash
# 1. 获取Provider列表
curl "http://20.2.140.226:8080/api/v1/acme/providers"

# 2. 创建账号（providerName: google，需要EAB）
curl -X POST http://20.2.140.226:8080/api/v1/acme/account/create \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "google",
    "email": "admin@example.com",
    "eabKid": "your_eab_kid_here",
    "eabHmacKey": "your_eab_hmac_key_here"
  }'
```

#### 场景3：切换默认账号

```bash
# 1. 创建新账号
curl -X POST http://20.2.140.226:8080/api/v1/acme/account/create \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "letsencrypt",
    "email": "backup@example.com",
    "eabKid": "",
    "eabHmacKey": ""
  }'

# 2. 设置新账号为默认（自动取消旧默认）
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/set-default \
  -H "Content-Type: application/json" \
  -d '{
    "providerId": 1,
    "accountId": 14
  }'

# 3. 禁用旧账号（现在可以禁用了）
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/disable \
  -H "Content-Type: application/json" \
  -d '{
    "id": 12
  }'
```

#### 场景4：安全删除账号

```bash
# 1. 检查账号是否为默认
curl "http://20.2.140.226:8080/api/v1/acme/accounts?page=1&pageSize=20"

# 2. 如果是默认账号，先设置其他账号为默认
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/set-default \
  -H "Content-Type: application/json" \
  -d '{
    "providerId": 1,
    "accountId": 14
  }'

# 3. 删除账号
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/delete \
  -H "Content-Type: application/json" \
  -d '{
    "id": 12
  }'
```

---

## 数据模型

### Provider

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | Provider ID |
| name | string | Provider名称 |
| directory_url | string | ACME目录URL |
| requires_eab | boolean | 是否需要EAB |
| status | string | 状态 |

### Account

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 账号ID |
| provider_id | int | Provider ID |
| email | string | 邮箱地址 |
| status | string | 状态（pending/active/disabled） |
| account_url | string | ACME账号URL |
| eab_kid | string | EAB Key ID |
| eab_hmac_key | string | EAB HMAC Key（不在响应中返回） |
| private_key | text | 私钥（不在响应中返回） |

### Provider Default

| 字段 | 类型 | 说明 |
|------|------|------|
| provider_id | int | Provider ID（主键） |
| account_id | int | 默认账号ID |

---

## 注意事项

### 安全性

1. 私钥（private_key）和EAB HMAC Key不会在任何响应中返回
2. 建议使用HTTPS传输敏感信息
3. EAB凭证应妥善保管，不要泄露

### 业务约束

1. 每个Provider只能有一个默认账号
2. 默认账号不能被禁用或删除
3. 被证书引用的账号不能删除
4. 同一Provider下，同一邮箱只能创建一个账号

### 最佳实践

1. 创建账号前先查询Provider列表，确认requiresEab字段
2. 对于需要EAB的Provider，确保提供有效的凭证
3. 删除账号前检查是否为默认账号或被引用
4. 使用分页查询避免一次性加载过多数据
5. 定期检查账号状态，及时处理pending状态的账号

---

## 附录

### Provider名称映射

| Provider Name | Provider ID | 说明 |
|---------------|-------------|------|
| letsencrypt | 1 | Let's Encrypt |
| google | 2 | Google Trust Services |

### 账号状态说明

| 状态 | 说明 |
|------|------|
| pending | 待确认（刚创建，等待ACME服务器确认） |
| active | 活跃（可用于证书申请） |
| disabled | 已禁用（不可用于证书申请） |

### 错误处理建议

1. **code: 2002**（参数验证失败）
   - 检查请求参数是否完整
   - 确认providerName拼写正确
   - 确认email格式正确

2. **code: 3002**（业务逻辑错误）
   - 禁用默认账号：先设置其他账号为默认
   - 删除默认账号：先设置其他账号为默认
   - 删除被引用账号：先删除相关证书请求或证书

3. **code: 5000**（服务器错误）
   - 检查服务器日志
   - 确认数据库连接正常
   - 联系系统管理员

---

文档版本：v1.0  
最后更新：2026-01-27  
维护者：CMDB开发团队
