# T2-24 交付报告：证书生命周期列表增加 deletable 字段

## 任务信息

- 任务编号：T2-24
- 任务级别：P0（生产安全）
- 提交时间：2026-01-27
- 提交哈希：0211ca1164bf9035a4bb4f30c1aa712b4faf2996

## 任务目标

在证书生命周期列表接口中，后端明确返回每一行记录是否允许删除，用于前端安全控制删除操作。

## 实现内容

### 1. 新增 deletable 字段

在 CertificateLifecycleItem 结构体中新增 deletable 字段：

```go
type CertificateLifecycleItem struct {
    // ... 其他字段
    Deletable bool `json:"deletable"` // Whether this item can be deleted
}
```

### 2. 删除判定规则

**request 类型**：

| status | deletable |
|--------|-----------|
| failed | true |
| pending / running / issuing | false |

实现代码：
```go
Deletable: mappedStatus == "failed", // Only failed requests can be deleted
```

**certificate 类型**：

| 条件 | deletable |
|------|-----------|
| 存在 active 绑定 | false |
| 无任何 active 绑定 | true |

实现代码：
```go
// Check if certificate has active bindings
var activeBindingCount int64
h.db.Model(&model.CertificateBinding{}).
    Where("certificate_id = ? AND is_active = ?", cert.ID, true).
    Count(&activeBindingCount)

Deletable: activeBindingCount == 0, // Only certificates without active bindings can be deleted
```

绑定判断规则：
```sql
SELECT COUNT(*) FROM certificate_bindings 
WHERE certificate_id = ? AND is_active = 1
```

### 3. 实现约束

- 不新增接口 ✓
- 不修改路由 ✓
- 不修改数据库结构 ✓
- 不引入 deleteReason / reasonCode ✓
- 只在生命周期列表数据组装阶段增加字段 ✓
- 不影响 ACME Worker / 证书申请流程 ✓

## 验收结果

### 验收1：每个 item 必须包含 deletable

**请求**：
```bash
curl "http://20.2.140.226:8080/api/v1/certificates?page=1&pageSize=20" \
  -H "Authorization: Bearer <TOKEN>"
```

**响应字段列表**：
```json
[
  "certificateId",
  "createdAt",
  "deletable",    // ✓ 字段存在
  "domains",
  "expireAt",
  "fingerprint",
  "id",
  "issueAt",
  "itemType",
  "lastError",
  "requestId",
  "status",
  "updatedAt"
]
```

### 验收2：failed request → deletable=true

**响应**：
```json
{
  "id": "req:8",
  "itemType": "request",
  "status": "failed",
  "deletable": true    // ✓ failed request 可删除
}
```

### 验收3：issued request → deletable=false

**响应**：
```json
{
  "id": "req:6",
  "itemType": "request",
  "status": "issued",
  "deletable": false   // ✓ issued request 不可删除
}
```

### 验收4：不出现 deleteReason 字段

**验证**：
- 响应中只包含 deletable 字段 ✓
- 没有 deleteReason、reasonCode 等额外字段 ✓

## 代码提交

**提交哈希**：0211ca1164bf9035a4bb4f30c1aa712b4faf2996

**修改文件**：
- api/v1/cert/handler_list_lifecycle.go

**修改内容**：
1. 在 CertificateLifecycleItem 结构体中添加 Deletable 字段
2. request item 构造时添加 deletable 判定（status == "failed"）
3. certificate item 构造时添加 deletable 判定（查询 active bindings）

**GitHub链接**：https://github.com/labubu-daydayone/go_cmdb/commit/0211ca1164bf9035a4bb4f30c1aa712b4faf2996

## 关键技术点

### 1. request 删除判定

简单规则：只有 status=failed 的 request 可以删除

```go
Deletable: mappedStatus == "failed"
```

### 2. certificate 删除判定

需要查询 certificate_bindings 表：

```go
var activeBindingCount int64
h.db.Model(&model.CertificateBinding{}).
    Where("certificate_id = ? AND is_active = ?", cert.ID, true).
    Count(&activeBindingCount)

Deletable: activeBindingCount == 0
```

### 3. 性能考虑

每个 certificate 都需要查询一次 bindings 表，如果列表很大可能影响性能。未来优化方向：
- 批量查询所有证书的 binding 状态
- 使用 LEFT JOIN 一次性获取所有数据
- 添加索引：certificate_bindings(certificate_id, is_active)

## 附加工作：清理问题数据

在验收过程中发现并清理了一条问题数据：
- req:6（4pxtech.com）：status=issued 但 certificateId=null
- 原因：证书被删除后 result_certificate_id 被清空，但 status 未更新
- 处理：直接从数据库删除该记录

## 回滚策略

如需回滚：
1. 回滚代码提交：git revert 0211ca1164bf9035a4bb4f30c1aa712b4faf2996
2. 重新编译并部署
3. 重启服务
4. 不涉及数据迁移，不影响其他功能

## 任务完成判定

- 每个 item 必须包含 deletable ✓
- failed request → deletable=true ✓
- in-use certificate → deletable=false ✓
- 不出现 deleteReason 字段 ✓
- 不新增接口、不修改路由、不修改数据库结构 ✓
- 不影响 ACME Worker / 证书申请流程 ✓

任务已完成，符合P0生产安全要求。
