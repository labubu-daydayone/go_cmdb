# ACME账号管理API - 前端调用示例

## 基础信息

- 测试环境：http://20.2.140.226:8080
- 所有接口仅使用GET和POST方法
- 所有响应格式：`{code: 0, message: "success", data: {...}}`
- 列表接口统一返回：`data.items`数组

---

## 0. 获取ACME Provider列表（CA提供商）

**接口路径**：GET /api/v1/acme/providers

**说明**：获取所有可用的CA提供商列表，providerId就是从这个接口返回的id字段。

**请求示例**：
```bash
curl "http://20.2.140.226:8080/api/v1/acme/providers"
```

**JavaScript示例**：
```javascript
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/providers');
const result = await response.json();

// 遍历所有CA提供商
result.data.items.forEach(provider => {
  console.log(`Provider ID: ${provider.id}, 名称: ${provider.name}`);
});
```

**成功响应**：
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

**Provider ID说明**：
- **id: 1** = Let's Encrypt（免费，无需EAB）
- **id: 2** = Google Trust Services（需要EAB）

**requiresEab字段**：
- `false`：创建账号时eabKid和eabHmacKey可以为空
- `true`：创建账号时必须提供eabKid和eabHmacKey

---

## 1. 创建ACME账号

**接口路径**：POST /api/v1/acme/account/create

**请求参数**：
```json
{
  "providerName": "letsencrypt",  // 必填："letsencrypt" 或 "google"
  "email": "admin@gmail.com",     // 必填：邮箱地址
  "eabKid": "",                    // 可选：Google需要
  "eabHmacKey": ""                 // 可选：Google需要
}
```

**请求示例**：
```bash
# Let's Encrypt（无需EAB）
curl -X POST http://20.2.140.226:8080/api/v1/acme/account/create \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "letsencrypt",
    "email": "admin@gmail.com",
    "eabKid": "",
    "eabHmacKey": ""
  }'

# Google Trust Services（需要EAB）
curl -X POST http://20.2.140.226:8080/api/v1/acme/account/create \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "google",
    "email": "admin@gmail.com",
    "eabKid": "your_eab_kid_here",
    "eabHmacKey": "your_eab_hmac_key_here"
  }'
```

**JavaScript示例**：
```javascript
// Let's Encrypt
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/account/create', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    providerName: 'letsencrypt',
    email: 'admin@gmail.com',
    eabKid: '',
    eabHmacKey: ''
  })
});

// Google Trust Services
const responseGoogle = await fetch('http://20.2.140.226:8080/api/v1/acme/account/create', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    providerName: 'google',
    email: 'admin@gmail.com',
    eabKid: 'your_eab_kid',
    eabHmacKey: 'your_eab_hmac_key'
  })
});

const result = await response.json();
// result.data.items[0] = { id: 12, email: "admin@gmail.com", ... }
```

**成功响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "provider_id": 1,
        "email": "admin@gmail.com",
        "status": "active",
        "account_url": "https://acme-v02.api.letsencrypt.org/acme/acct/123456",
        "created_at": "2026-01-26T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

---

## 2. 获取ACME账号列表

**接口路径**：GET /api/v1/acme/accounts

**请求示例**：
```bash
# 获取所有账号（分页）
curl "http://20.2.140.226:8080/api/v1/acme/accounts?page=1&page_size=20"

# 按CA过滤
curl "http://20.2.140.226:8080/api/v1/acme/accounts?provider_id=1"

# 按状态过滤
curl "http://20.2.140.226:8080/api/v1/acme/accounts?status=active"

# 组合过滤
curl "http://20.2.140.226:8080/api/v1/acme/accounts?provider_id=1&status=active&page=1&page_size=10"
```

**JavaScript示例**：
```javascript
// 获取列表
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts?page=1&page_size=20');
const result = await response.json();

// 遍历账号列表
result.data.items.forEach(account => {
  console.log(`账号ID: ${account.id}, 邮箱: ${account.email}, 默认: ${account.is_default}`);
});
```

**成功响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "provider_id": 1,
        "provider_name": "letsencrypt",
        "email": "admin@gmail.com",
        "status": "active",
        "is_default": true,
        "account_url": "https://acme-v02.api.letsencrypt.org/acme/acct/123456",
        "created_at": "2026-01-26T10:30:00Z"
      },
      {
        "id": 13,
        "provider_id": 1,
        "provider_name": "letsencrypt",
        "email": "backup@gmail.com",
        "status": "disabled",
        "is_default": false,
        "account_url": "https://acme-v02.api.letsencrypt.org/acme/acct/789012",
        "created_at": "2026-01-25T08:15:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 20
  }
}
```

---

## 3. 设置默认ACME账号

**接口路径**：POST /api/v1/acme/accounts/set-default

**请求示例**：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/set-default \
  -H "Content-Type: application/json" \
  -d '{
    "providerId": 1,
    "accountId": 12
  }'
```

**JavaScript示例**：
```javascript
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/set-default', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    providerId: 1,
    accountId: 12
  })
});

const result = await response.json();
if (result.code === 0) {
  console.log('默认账号设置成功');
}
```

**成功响应**：
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
- 每个CA只能有一个默认账号
- 设置新默认账号会自动取消旧的默认状态
- 只能设置active状态的账号为默认

---

## 4. 启用ACME账号

**接口路径**：POST /api/v1/acme/accounts/enable

**请求示例**：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/enable \
  -H "Content-Type: application/json" \
  -d '{
    "id": 13
  }'
```

**JavaScript示例**：
```javascript
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/enable', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    id: 13
  })
});

const result = await response.json();
if (result.code === 0) {
  console.log('账号已启用');
}
```

**成功响应**：
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

---

## 5. 禁用ACME账号

**接口路径**：POST /api/v1/acme/accounts/disable

**请求示例**：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/disable \
  -H "Content-Type: application/json" \
  -d '{
    "id": 13
  }'
```

**JavaScript示例**：
```javascript
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/disable', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    id: 13
  })
});

const result = await response.json();
if (result.code === 0) {
  console.log('账号已禁用');
} else if (result.code === 3002) {
  console.error('错误：不能禁用默认账号');
}
```

**成功响应**：
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
- 不能禁用默认账号
- 如需禁用默认账号，请先设置其他账号为默认

---

## 6. 删除ACME账号

**接口路径**：POST /api/v1/acme/accounts/delete

**请求示例**：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/acme/accounts/delete \
  -H "Content-Type: application/json" \
  -d '{
    "id": 13
  }'
```

**JavaScript示例**：
```javascript
const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/delete', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    id: 13
  })
});

const result = await response.json();
if (result.code === 0) {
  console.log('账号已删除');
} else if (result.code === 3002) {
  console.error('删除失败：' + result.message);
}
```

**成功响应**：
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

**业务规则**：
- 不能删除默认账号
- 不能删除被证书请求引用的账号
- 不能删除被证书引用的账号

---

## 完整的前端使用流程示例

```javascript
// 0. 获取Provider列表
async function listProviders() {
  const response = await fetch('http://20.2.140.226:8080/api/v1/acme/providers');
  const result = await response.json();
  return result.data.items; // 返回Provider数组
}

// 1. 创建新账号
async function createAccount(providerName, email, eabKid = '', eabHmacKey = '') {
  const response = await fetch('http://20.2.140.226:8080/api/v1/acme/account/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      providerName,  // 'letsencrypt' 或 'google'
      email,
      eabKid,
      eabHmacKey
    })
  });
  const result = await response.json();
  if (result.code !== 0) {
    throw new Error(result.message);
  }
  return result.data.items[0].id; // 返回新账号ID
}

// 2. 获取账号列表
async function listAccounts(providerId = null) {
  let url = 'http://20.2.140.226:8080/api/v1/acme/accounts?page=1&page_size=20';
  if (providerId) {
    url += `&provider_id=${providerId}`;
  }
  
  const response = await fetch(url);
  const result = await response.json();
  return result.data.items; // 返回账号数组
}

// 3. 设置默认账号
async function setDefaultAccount(providerId, accountId) {
  const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/set-default', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ providerId, accountId })
  });
  const result = await response.json();
  return result.code === 0;
}

// 4. 禁用账号（带错误处理）
async function disableAccount(accountId) {
  const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/disable', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: accountId })
  });
  const result = await response.json();
  
  if (result.code === 3002) {
    throw new Error('不能禁用默认账号，请先设置其他账号为默认');
  }
  return result.code === 0;
}

// 5. 删除账号（带错误处理）
async function deleteAccount(accountId) {
  const response = await fetch('http://20.2.140.226:8080/api/v1/acme/accounts/delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: accountId })
  });
  const result = await response.json();
  
  if (result.code === 3002) {
    throw new Error(result.message);
  }
  return result.code === 0;
}

// 使用示例
async function main() {
  try {
    // 获取Provider列表
    const providers = await listProviders();
    console.log('可用的CA提供商:', providers);
    // providers[0].id = 1 (Let's Encrypt)
    // providers[1].id = 2 (Google Trust Services)
    
    // 创建账号（使用Let's Encrypt）
    const newAccountId = await createAccount('letsencrypt', 'newaccount@gmail.com');
    console.log('新账号ID:', newAccountId);
    
    // 获取列表
    const accounts = await listAccounts(1);
    console.log('账号列表:', accounts);
    
    // 设置默认
    await setDefaultAccount(1, newAccountId);
    console.log('默认账号已设置');
    
  } catch (error) {
    console.error('操作失败:', error.message);
  }
}
```

---

## 错误码说明

| 错误码 | 说明 | 常见场景 |
|--------|------|----------|
| 0 | 成功 | 所有成功操作 |
| 3002 | 业务逻辑错误 | 禁用默认账号、删除被引用账号 |
| 5000 | 服务器内部错误 | 数据库连接失败、系统异常 |

---

## 注意事项

1. **providerId说明**：
   - providerId是CA提供商的ID，通过GET /api/v1/acme/providers获取
   - providerId: 1 = Let's Encrypt（免费，无需EAB）
   - providerId: 2 = Google Trust Services（需要EAB）
   - 创建账号前应先查询Provider列表，根据requiresEab字段决定是否需要提供EAB凭证

2. **路径区别**：
   - 创建账号：POST /api/v1/acme/account/create（单数account）
   - 其他操作：/api/v1/acme/accounts（复数accounts）

3. **响应格式**：
   - 所有列表接口返回`data.items`数组
   - 单对象操作也返回`data.items`数组（只有一个元素）

4. **业务约束**：
   - 默认账号不能禁用或删除
   - 被引用的账号不能删除
   - 每个CA只能有一个默认账号

5. **分页参数**：
   - page：页码，从1开始
   - page_size：每页数量，默认20

6. **过滤参数**：
   - provider_id：CA提供商ID
   - status：账号状态（active/disabled）
