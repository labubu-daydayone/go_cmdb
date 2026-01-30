# 交付报告：Bug Fix - 回源地址 `enabled: false` 未生效

**任务目标**：修复 `POST /api/v1/origin-groups/addresses/upsert` 接口在提交 `enabled: false` 时，数据库中仍然保存为 `true` 的问题。

## 1. 问题根源分析

经过排查，问题由两个因素共同导致：

1.  **数据库表结构**：`origin_group_addresses` 表的 `enabled` 字段（`tinyint(1)`）设置了 `DEFAULT '1'`。
2.  **GORM 零值问题**：当使用 `db.Create(&addresses)` 进行批量插入时，GORM 会忽略 Go 结构体中的零值字段。对于 `bool` 类型，`false` 就是其零值。因此，当请求中 `enabled` 为 `false` 时，GORM 在生成 SQL 时会忽略该字段，导致数据库应用了 `DEFAULT '1'`，最终数据被错误地保存为 `true`。

多次尝试使用 `tx.Select()` 指定字段未能解决此问题，这表明 GORM 在批量创建（`Create(&slice)`) 模式下处理零值和数据库默认值的行为较为复杂。

## 2. 解决方案

为确保 `enabled` 字段的值被精确地写入数据库，最终采用了最直接可靠的方案：**使用原生 SQL `INSERT` 语句**。

在 `AddressesUpsert` 处理器中，将原来的 `tx.Create(&addresses)` 批量创建逻辑，修改为在循环中对每个地址执行一次原生 `INSERT`。这可以完全绕过 GORM 的 ORM 零值处理逻辑，保证了请求中的 `enabled` 值被忠实地传递给数据库。

### 2.1. 修改文件清单

- `api/v1/origin_groups/handler.go`

### 2.2. 核心代码变更

```go
// ...
// 创建新地址 - 逐条插入以确保 enabled 字段正确保存
for _, item := range req.Items {
    addr := model.OriginGroupAddress{
        OriginGroupID: req.OriginGroupID,
        Address:       item.Address,
        Role:          item.Role,
        Weight:        item.Weight,
        Enabled:       item.Enabled,
        Protocol:      model.OriginProtocolHTTP,
    }
    // 使用原生 SQL 确保 enabled 字段被正确设置
    if err := tx.Exec(
        "INSERT INTO origin_group_addresses (origin_group_id, address, role, weight, enabled, protocol, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())",
        addr.OriginGroupID, addr.Address, addr.Role, addr.Weight, addr.Enabled, addr.Protocol,
    ).Error; err != nil {
        tx.Rollback()
        httpx.FailErr(c, httpx.ErrDatabaseError("failed to create address", err))
        return
    }
}
// ...
```

## 3. 验收测试

### 3.1. 测试请求

发送一个包含 `enabled: false` 和 `enabled: true` 的请求。

- **请求**：
```bash
TOKEN=$(curl -s -X POST http://20.2.140.226:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-groups/addresses/upsert" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{
    "originGroupId": 2,
    "items": [
        {
            "address": "10.0.0.1:80",
            "role": "primary",
            "weight": 10,
            "enabled": false
        },
        {
            "address": "10.0.0.2:80",
            "role": "primary",
            "weight": 20,
            "enabled": true
        }
    ]
}'
```

### 3.2. 测试响应

接口返回的数据现在能正确反映提交的状态。

- **响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "item": {
            "originGroupId": 2,
            "items": [
                {
                    "id": 57,
                    "address": "10.0.0.1:80",
                    "role": "primary",
                    "weight": 10,
                    "enabled": false,
                    "createdAt": "2026-01-30T19:19:43+08:00",
                    "updatedAt": "2026-01-30T19:19:43+08:00"
                },
                {
                    "id": 58,
                    "address": "10.0.0.2:80",
                    "role": "primary",
                    "weight": 20,
                    "enabled": true,
                    "createdAt": "2026-01-30T19:19:43+08:00",
                    "updatedAt": "2026-01-30T19:19:43+08:00"
                }
            ]
        }
    }
}
```

### 3.3. 数据库验证

查询数据库，确认 `enabled` 字段已正确写入 `0`。

- **查询**：
```sql
SELECT id, address, enabled FROM origin_group_addresses WHERE id IN (57, 58);
```

- **结果**：
```
+----+-------------+---------+
| id | address     | enabled |
+----+-------------+---------+
| 57 | 10.0.0.1:80 |       0 |
| 58 | 10.0.0.2:80 |       1 |
+----+-------------+---------+
```

## 4. 结论

Bug 已成功修复。`addresses/upsert` 接口现在可以正确处理 `enabled` 字段的 `false` 值。代码已提交至 GitHub 仓库。

## 5. 回滚方案

如需回滚，可 revert 本次修复的 commit。

```bash
# commit hash for the fix
COMMIT_HASH="c2a50a8"

# Revert the commit
git revert $COMMIT_HASH

# Re-deploy the service
# ... (compile and restart service)
```
