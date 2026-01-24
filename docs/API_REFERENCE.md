# API Reference

## 基本信息

**Base URL:** `http://20.2.140.226:8080/api/v1`

**认证方式:** JWT Bearer Token

**Content-Type:** `application/json`

**默认账号:**
- 用户名: `admin`
- 密码: `admin123`
- 角色: `admin`

## 认证流程

### 1. 登录获取Token

```bash
curl -X POST http://20.2.140.226:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

响应示例:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expireAt": "2026-01-25T13:52:42+08:00",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 2. 使用Token访问受保护的API

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/me
```

## 响应格式

所有API响应遵循统一格式:

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误码说明:**
- `0`: 成功
- `1001`: 认证失败（缺少token或token无效）
- `1002`: 凭据无效（用户名或密码错误）
- `1003`: 权限不足
- `2001`: 参数错误
- `2002`: 资源不存在
- `2003`: 资源已存在
- `5001`: 数据库错误
- `5002`: 内部服务错误

---

## API端点列表

### 公开端点（无需认证）

#### 1. Ping - 健康检查

**GET** `/ping`

测试服务是否运行。

**请求示例:**
```bash
curl http://20.2.140.226:8080/api/v1/ping
```

**响应示例:**
```json
{
  "code": 0,
  "message": "pong"
}
```

#### 2. 用户登录

**POST** `/auth/login`

用户登录获取JWT token。

**请求体:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应示例:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expireAt": "2026-01-25T13:52:42+08:00",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

---

### 受保护端点（需要认证）

所有以下端点都需要在请求头中包含: `Authorization: Bearer YOUR_TOKEN`

#### 3. 获取当前用户信息

**GET** `/me`

获取当前登录用户的信息。

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/me
```

**响应示例:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

---

## 节点管理 (Nodes)

### 4. 节点列表

**GET** `/nodes`

查询节点列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (active/inactive)

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://20.2.140.226:8080/api/v1/nodes?page=1&pageSize=20"
```

### 5. 创建节点

**POST** `/nodes/create`

创建新节点。

**请求体:**
```json
{
  "name": "edge-01",
  "main_ip": "192.168.1.100",
  "node_group_id": 1,
  "status": "active"
}
```

### 6. 更新节点

**POST** `/nodes/update`

更新节点信息。

**请求体:**
```json
{
  "id": 1,
  "name": "edge-01-updated",
  "status": "inactive"
}
```

### 7. 删除节点

**POST** `/nodes/delete`

删除节点（批量）。

**请求体:**
```json
{
  "ids": [1, 2, 3]
}
```

### 8. 添加子IP

**POST** `/nodes/sub-ips/add`

为节点添加子IP。

**请求体:**
```json
{
  "node_id": 1,
  "ips": ["192.168.1.101", "192.168.1.102"]
}
```

### 9. 删除子IP

**POST** `/nodes/sub-ips/delete`

删除节点的子IP。

**请求体:**
```json
{
  "ids": [1, 2]
}
```

### 10. 切换子IP状态

**POST** `/nodes/sub-ips/toggle`

启用/禁用子IP。

**请求体:**
```json
{
  "id": 1,
  "enabled": true
}
```

---

## 节点分组 (Node Groups)

### 11. 节点分组列表

**GET** `/node-groups`

查询节点分组列表。

### 12. 创建节点分组

**POST** `/node-groups/create`

创建新节点分组。

**请求体:**
```json
{
  "name": "华东节点",
  "description": "华东地区边缘节点"
}
```

### 13. 更新节点分组

**POST** `/node-groups/update`

更新节点分组信息。

### 14. 删除节点分组

**POST** `/node-groups/delete`

删除节点分组（批量）。

---

## 线路分组 (Line Groups)

### 15. 线路分组列表

**GET** `/line-groups`

查询线路分组列表。

### 16. 创建线路分组

**POST** `/line-groups/create`

创建新线路分组。

**请求体:**
```json
{
  "name": "电信线路",
  "domain_id": 1,
  "cname": "cdn.example.com"
}
```

### 17. 更新线路分组

**POST** `/line-groups/update`

更新线路分组信息。

### 18. 删除线路分组

**POST** `/line-groups/delete`

删除线路分组（批量）。

---

## 源站分组 (Origin Groups)

### 19. 源站分组列表

**GET** `/origin-groups`

查询源站分组列表。

### 20. 创建源站分组

**POST** `/origin-groups/create`

创建新源站分组。

**请求体:**
```json
{
  "name": "主源站组",
  "description": "主要源站服务器组"
}
```

### 21. 更新源站分组

**POST** `/origin-groups/update`

更新源站分组信息。

### 22. 删除源站分组

**POST** `/origin-groups/delete`

删除源站分组（批量）。

---

## 源站管理 (Origins)

### 23. 从分组创建源站

**POST** `/origins/create-from-group`

从源站分组创建源站配置。

### 24. 手动创建源站

**POST** `/origins/create-manual`

手动创建源站配置。

**请求体:**
```json
{
  "website_id": 1,
  "addresses": [
    {
      "role": "master",
      "protocol": "https",
      "address": "origin.example.com:443",
      "weight": 100,
      "enabled": true
    }
  ]
}
```

### 25. 更新源站

**POST** `/origins/update`

更新源站配置。

### 26. 删除源站

**POST** `/origins/delete`

删除源站配置。

---

## 网站管理 (Websites)

### 27. 网站列表

**GET** `/websites`

查询网站列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (active/inactive)

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://20.2.140.226:8080/api/v1/websites?page=1&pageSize=20"
```

**响应示例:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "line_group_id": 1,
        "line_group_name": "电信线路",
        "cache_rule_id": 0,
        "origin_mode": "group",
        "origin_group_id": 1,
        "origin_group_name": "主源站组",
        "status": "active",
        "domains": ["www.example.com"],
        "primary_domain": "www.example.com",
        "cname": "cdn.example.com",
        "https_enabled": false,
        "created_at": "2026-01-24 10:00:00",
        "updated_at": "2026-01-24 10:00:00"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 28. 获取网站详情

**GET** `/websites/:id`

获取指定网站的详细信息。

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/websites/1
```

### 29. 创建网站

**POST** `/websites/create`

创建新网站。

**请求体:**
```json
{
  "line_group_id": 1,
  "cache_rule_id": 0,
  "origin_mode": "group",
  "origin_group_id": 1,
  "domains": ["www.example.com", "example.com"],
  "https": {
    "enabled": true,
    "force_redirect": true,
    "hsts": false,
    "cert_mode": "select",
    "certificate_id": 1
  }
}
```

### 30. 更新网站

**POST** `/websites/update`

更新网站配置。

**请求体:**
```json
{
  "id": 1,
  "status": "inactive"
}
```

### 31. 删除网站

**POST** `/websites/delete`

删除网站（批量）。

**请求体:**
```json
{
  "ids": [1, 2, 3]
}
```

---

## Agent任务管理 (Agent Tasks)

### 32. Agent任务列表

**GET** `/agent-tasks`

查询Agent任务列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (pending/running/success/failed)

### 33. 获取Agent任务详情

**GET** `/agent-tasks/:id`

获取指定Agent任务的详细信息。

### 34. 创建Agent任务

**POST** `/agent-tasks/create`

创建新Agent任务。

### 35. 重试Agent任务

**POST** `/agent-tasks/retry`

重试失败的Agent任务。

**请求体:**
```json
{
  "id": 1
}
```

---

## Agent身份管理 (Agent Identities) - 仅管理员

### 36. Agent身份列表

**GET** `/agent-identities`

查询Agent身份列表（仅管理员）。

### 37. 创建Agent身份

**POST** `/agent-identities/create`

创建新Agent身份（仅管理员）。

### 38. 撤销Agent身份

**POST** `/agent-identities/revoke`

撤销Agent身份（仅管理员）。

---

## 配置管理 (Config)

### 39. 应用配置

**POST** `/config/apply`

应用配置到节点。

**请求体:**
```json
{
  "target": "cdn",
  "node_ids": [1, 2, 3]
}
```

### 40. 配置版本列表

**GET** `/config/versions`

查询配置版本列表。

### 41. 获取配置版本详情

**GET** `/config/versions/:version`

获取指定配置版本的详细信息。

---

## DNS管理 (DNS)

### 42. DNS记录列表

**GET** `/dns/records`

查询DNS记录列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (pending/success/failed)

### 43. 获取DNS记录详情

**GET** `/dns/records/:id`

获取指定DNS记录的详细信息。

### 44. 创建DNS记录

**POST** `/dns/records/create`

创建新DNS记录。

**请求体:**
```json
{
  "domain_id": 1,
  "type": "A",
  "name": "www",
  "value": "192.168.1.100",
  "ttl": 600
}
```

### 45. 删除DNS记录

**POST** `/dns/records/delete`

删除DNS记录。

**请求体:**
```json
{
  "id": 1
}
```

### 46. 重试DNS记录

**POST** `/dns/records/retry`

重试失败的DNS记录。

**请求体:**
```json
{
  "id": 1
}
```

---

## ACME证书管理 (ACME)

### 47. 创建ACME账户

**POST** `/acme/account/create`

创建新ACME账户。

**请求体:**
```json
{
  "provider_id": 1,
  "email": "admin@example.com"
}
```

### 48. 请求证书

**POST** `/acme/certificate/request`

请求新证书。

**请求体:**
```json
{
  "domains": ["www.example.com", "example.com"],
  "acme_account_id": 1,
  "dns_provider_id": 1
}
```

### 49. 重试证书请求

**POST** `/acme/certificate/retry`

重试失败的证书请求。

**请求体:**
```json
{
  "id": 1
}
```

### 50. 证书请求列表

**GET** `/acme/certificate/requests`

查询证书请求列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (pending/success/failed)

### 51. 获取证书请求详情

**GET** `/acme/certificate/requests/:id`

获取指定证书请求的详细信息。

---

## 证书续期管理 (Certificate Renewal)

### 52. 获取续期候选证书

**GET** `/certificates/renewal/candidates`

获取需要续期的证书列表。

### 53. 触发证书续期

**POST** `/certificates/renewal/trigger`

触发证书续期。

**请求体:**
```json
{
  "certificate_id": 1
}
```

### 54. 禁用自动续期

**POST** `/certificates/renewal/disable-auto`

禁用证书的自动续期。

**请求体:**
```json
{
  "certificate_id": 1
}
```

---

## 证书覆盖管理 (Certificate Coverage)

### 55. 获取证书关联的网站

**GET** `/certificates/:id/websites`

获取证书关联的所有网站。

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/certificates/1/websites
```

### 56. 获取网站的候选证书

**GET** `/websites/:id/certificates/candidates`

获取网站可用的候选证书列表。

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/websites/1/certificates/candidates
```

---

## 风险管理 (Risks)

### 57. 风险列表

**GET** `/risks`

查询风险列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20
- `status` (可选): 过滤状态 (open/resolved)
- `severity` (可选): 过滤严重程度 (critical/high/medium/low)

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://20.2.140.226:8080/api/v1/risks?page=1&pageSize=20&status=open"
```

### 58. 解决风险

**POST** `/risks/:id/resolve`

标记风险为已解决。

**请求示例:**
```bash
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/risks/1/resolve
```

### 59. 获取网站的风险

**GET** `/websites/:id/risks`

获取指定网站的所有风险。

### 60. 获取证书的风险

**GET** `/certificates/:id/risks`

获取指定证书的所有风险。

### 61. HTTPS预检查

**POST** `/websites/:id/precheck/https`

对网站进行HTTPS配置预检查。

**请求示例:**
```bash
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/websites/1/precheck/https
```

---

## 发布任务管理 (Releases)

### 62. 发布任务列表

**GET** `/releases`

查询发布任务列表。

**查询参数:**
- `page` (可选): 页码，默认1
- `pageSize` (可选): 每页数量，默认20，最大100
- `status` (可选): 过滤状态 (pending/running/success/failed/paused)

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://20.2.140.226:8080/api/v1/releases?page=1&pageSize=20&status=running"
```

**响应示例:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "type": "apply_config",
        "target": "cdn",
        "version": 123456,
        "status": "running",
        "totalNodes": 5,
        "successNodes": 2,
        "failedNodes": 1,
        "skippedNodes": 0,
        "currentBatch": 2,
        "createdAt": "2026-01-24T10:00:00+08:00",
        "updatedAt": "2026-01-24T10:05:00+08:00"
      }
    ],
    "total": 10,
    "page": 1,
    "pageSize": 20
  }
}
```

### 63. 创建发布任务

**POST** `/releases`

创建新发布任务。

**请求体:**
```json
{
  "type": "apply_config",
  "target": "cdn",
  "version": 123456,
  "node_ids": [1, 2, 3, 4, 5],
  "batch_size": 2,
  "batch_interval": 60
}
```

### 64. 获取发布任务详情

**GET** `/releases/:id`

获取指定发布任务的详细信息（含batch分组和node明细）。

**请求示例:**
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://20.2.140.226:8080/api/v1/releases/1
```

**响应示例:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "release": {
      "id": 1,
      "type": "apply_config",
      "target": "cdn",
      "version": 123456,
      "status": "running",
      "totalNodes": 5,
      "successNodes": 2,
      "failedNodes": 1,
      "skippedNodes": 0,
      "currentBatch": 2,
      "createdAt": "2026-01-24T10:00:00+08:00",
      "updatedAt": "2026-01-24T10:05:00+08:00"
    },
    "batches": [
      {
        "batch": 1,
        "nodes": [
          {
            "nodeId": 1,
            "nodeName": "edge-01",
            "status": "success",
            "errorMsg": "",
            "startedAt": "2026-01-24T10:00:00+08:00",
            "finishedAt": "2026-01-24T10:01:00+08:00"
          },
          {
            "nodeId": 2,
            "nodeName": "edge-02",
            "status": "failed",
            "errorMsg": "Connection timeout",
            "startedAt": "2026-01-24T10:00:00+08:00",
            "finishedAt": "2026-01-24T10:01:30+08:00"
          }
        ]
      },
      {
        "batch": 2,
        "nodes": [
          {
            "nodeId": 3,
            "nodeName": "edge-03",
            "status": "running",
            "errorMsg": "",
            "startedAt": "2026-01-24T10:02:00+08:00",
            "finishedAt": null
          }
        ]
      }
    ]
  }
}
```

---

## WebSocket实时同步 (Socket.IO)

### 65. WebSocket连接

**连接地址:** `ws://20.2.140.226:8080/socket.io/`

**认证方式:** JWT Token（通过query参数或auth对象传递）

**支持的事件:**

#### 客户端发送事件

**request:websites** - 请求网站列表

发送数据:
```json
{
  "lastEventId": 0
}
```

- `lastEventId`: 上次接收到的事件ID，0表示请求全量数据

#### 服务端推送事件

**connected** - 连接确认

接收数据:
```json
{
  "ok": true
}
```

**websites:initial** - 初始全量数据

接收数据:
```json
{
  "items": [ ...website_list_items... ],
  "total": 123,
  "version": 0,
  "lastEventId": 1024
}
```

**websites:update** - 增量更新

接收数据:
```json
{
  "eventId": 1025,
  "type": "add|update|delete",
  "data": {
    "id": 1001
  }
}
```

### Node.js客户端示例

```javascript
const io = require('socket.io-client');

// 获取JWT token（从登录接口）
const token = 'YOUR_JWT_TOKEN_HERE';

// 连接Socket.IO服务器
const socket = io('http://20.2.140.226:8080', {
  auth: { token }
});

// 连接成功
socket.on('connect', () => {
  console.log('Connected:', socket.id);
});

// 收到connected确认
socket.on('connected', (data) => {
  console.log('Connected confirmation:', data);
  
  // 请求websites列表
  socket.emit('request:websites', { lastEventId: 0 });
});

// 收到初始全量
socket.on('websites:initial', (data) => {
  console.log('Websites initial:', data);
});

// 收到增量更新
socket.on('websites:update', (data) => {
  console.log('Websites update:', data);
});

// 连接错误
socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message);
});

// 断开连接
socket.on('disconnect', (reason) => {
  console.log('Disconnected:', reason);
});
```

### 浏览器客户端示例

```html
<!DOCTYPE html>
<html>
<head>
  <title>WebSocket Test</title>
  <script src="https://cdn.socket.io/4.5.4/socket.io.min.js"></script>
</head>
<body>
  <h1>WebSocket Test</h1>
  <div id="status">Disconnected</div>
  <div id="messages"></div>

  <script>
    const token = 'YOUR_JWT_TOKEN_HERE';
    
    const socket = io('http://20.2.140.226:8080', {
      auth: { token }
    });

    socket.on('connect', () => {
      document.getElementById('status').textContent = 'Connected: ' + socket.id;
    });

    socket.on('connected', (data) => {
      console.log('Connected confirmation:', data);
      socket.emit('request:websites', { lastEventId: 0 });
    });

    socket.on('websites:initial', (data) => {
      console.log('Websites initial:', data);
      document.getElementById('messages').innerHTML += '<p>Received initial: ' + data.total + ' websites</p>';
    });

    socket.on('websites:update', (data) => {
      console.log('Websites update:', data);
      document.getElementById('messages').innerHTML += '<p>Update: ' + data.type + ' (eventId: ' + data.eventId + ')</p>';
    });

    socket.on('connect_error', (error) => {
      document.getElementById('status').textContent = 'Error: ' + error.message;
    });

    socket.on('disconnect', (reason) => {
      document.getElementById('status').textContent = 'Disconnected: ' + reason;
    });
  </script>
</body>
</html>
```

---

## 测试示例

### 完整的API调用流程

```bash
# 1. 登录获取token
TOKEN=$(curl -s -X POST http://20.2.140.226:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')

echo "Token: $TOKEN"

# 2. 获取当前用户信息
curl -H "Authorization: Bearer $TOKEN" \
  http://20.2.140.226:8080/api/v1/me

# 3. 查询网站列表
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/websites?page=1&pageSize=20"

# 4. 查询发布任务列表
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/releases?page=1&pageSize=20"

# 5. 查询风险列表
curl -H "Authorization: Bearer $TOKEN" \
  "http://20.2.140.226:8080/api/v1/risks?page=1&pageSize=20&status=open"
```

---

## 附录

### A. 数据模型说明

#### 节点状态 (Node Status)
- `active`: 活跃
- `inactive`: 停用

#### 发布任务类型 (Release Type)
- `apply_config`: 应用配置

#### 发布任务状态 (Release Status)
- `pending`: 等待中
- `running`: 运行中
- `success`: 成功
- `failed`: 失败
- `paused`: 已暂停

#### 发布任务节点状态 (Release Task Node Status)
- `pending`: 等待中
- `running`: 运行中
- `success`: 成功
- `failed`: 失败
- `skipped`: 已跳过

#### 风险严重程度 (Risk Severity)
- `critical`: 严重
- `high`: 高
- `medium`: 中
- `low`: 低

#### 风险状态 (Risk Status)
- `open`: 未解决
- `resolved`: 已解决

### B. 常见问题

**Q: 如何处理token过期？**

A: Token过期后API会返回401错误，需要重新调用登录接口获取新token。

**Q: 如何实现分页查询？**

A: 使用page和pageSize参数，例如: `/api/v1/websites?page=2&pageSize=50`

**Q: WebSocket连接失败怎么办？**

A: 检查token是否有效，确保使用正确的连接方式（auth对象或query参数）。

**Q: 如何批量操作？**

A: 大多数删除接口支持批量操作，传入ids数组即可。

---

## 更新日志

### 2026-01-24
- 初始版本
- 包含所有已实现的API端点
- 添加WebSocket实时同步文档
- 添加完整的请求/响应示例

---

**文档版本:** v1.0.0
**最后更新:** 2026-01-24
**维护者:** 开发团队
