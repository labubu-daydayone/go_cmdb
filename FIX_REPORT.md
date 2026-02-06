# Bug 修复与验收报告：origin_set 创建与 bind-website 幂等性

## 一、问题描述

1.  **`origin-sets/create` 缺陷**：通过 `origin_group_id` 创建回源配置（`origin_set`）时，未从 `origin_group_addresses` 表中拉取回源地址列表，导致生成的 `snapshot_json.addresses` 字段为空数组。
2.  **`bind-website` 幂等性缺陷**：将网站绑定到回源配置时，用于判断配置是否变更的 `contentHash` 计算包含了 `originSetId` 等非内容字段，或存在因 `map` 序列化顺序不稳定导致 `hash` 值变化的问题。这导致即使回源地址列表完全相同，仅切换 `origin_set` 也会错误地生成新的发布任务（`release_task`），破坏了幂等性。

## 二、修复方案

### 1. `origin-sets/create` 接口修复

在 `api/v1/origin_sets/handler.go` 的 `Create` 方法中，增加核心处理逻辑：

-   **校验主地址**：创建时，强制检查 `origin_group_addresses` 中是否存在至少一个 `enabled=true` 且 `role='primary'` 的地址。若不存在，则返回 `code=2001`，拒绝创建，避免产生无效的空快照。
-   **地址数据填充**：从 `origin_group_addresses` 表中查询所有与 `originGroupId` 关联的有效地址，并将其序列化后写入 `origin_set_items.snapshot_json` 字段。

### 2. `bind-website` 幂等性修复

在 `internal/service/release_task_service.go` 的 `CreateWebsiteReleaseTask` 方法中，重构 `contentHash` 计算逻辑：

-   **地址列表校验**：在绑定操作的起始阶段，增加对 `origin_set` 的 `snapshot_json.addresses` 的校验。如果地址列表为空，则直接返回 `code=2001` 拒绝操作。
-   **`contentHash` 计算标准化**：
    -   **限定哈希内容**：`contentHash` 的计算范围严格限定为回源地址列表（`origins`），移除所有无关 ID（如 `originSetId`, `websiteId`）和元数据。
    -   **使用结构体保证顺序**：将用于哈希的 `payload` 从 `map[string]interface{}` 改为固定字段顺序的 `struct`，彻底解决 `map` 迭代顺序不确定性问题。
    -   **稳定排序**：在哈希前，对 `origins` 列表进行稳定排序（按 `address`, `role`, `weight` 等字段），确保内容相同但顺序不同的地址列表也能生成一致的 `hash` 值。

### 3. 调试与日志增强

为快速定位问题，引入了以下调试机制：

-   **统一日志**：使用 `util.DebugLog` 辅助函数，将调试日志同时输出到 `stdout` 和固定的日志文件 `/opt/go_cmdb/var/debug/web_release_debug.log`。
-   **关键点打点**：在 `bind-website` 链路的关键位置（Handler 入口、Service 调用前后、Hash 计算前后）增加了带有唯一 `traceId` 的日志打点。
-   **Payload 落盘**：在计算 `contentHash` 时，将序列化后的 `payload` 内容写入 `/opt/go_cmdb/var/debug/hashpayload_*.json` 文件，便于直接 `diff` 对比两次请求的内容差异。

## 三、验收过程

使用 `curl` 命令对修复后的服务进行端到端测试，IP 地址为 `20.2.140.226`。

### 1. 创建回源配置 (origin_set)

**操作**：基于 `origin_group_id=2` 创建一个新的 `origin_set`（`final-test-set-A1`）。

```shell
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc3MDQwNDQ2OCwiaWF0IjoxNzcwMzE4MDY4fQ.6uWvP03CHE0qq6b456pIsSuNtZ8-0zd5gXtgzYzTK_Y"
curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-sets/create" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{"name":"final-test-set-A1","description":"final test A1","originGroupId":2}'
```

**返回**：

```json
{"code":0,"message":"success","data":{"item":{"id":12,"name":"final-test-set-A1","description":"final test A1","status":"active","originGroupId":2,"createdAt":"2026-02-06T04:15:27+08:00","updatedAt":"2026-02-06T04:15:27+08:00"}}}
```

**验证**：查询 `origin_set_items` 表，确认 `snapshot_json` 包含来自 `origin_group_id=2` 的地址，修复生效。

### 2. 首次绑定网站

**操作**：将 `website_id=41` 绑定到新创建的 `origin_set_id=12`。

```shell
curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-sets/bind-website" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{"websiteId":41,"originSetId":12}'
```

**返回**：

```json
{"code":0,"message":"success","data":{"item":{"originSetId":12,"skipReason":"","taskCreated":true,"taskId":11,"websiteId":41}}}
```

**验证**：`taskCreated` 为 `true`，生成了新的 `taskId=11`，符合预期。

### 3. 幂等性测试：切换到相同内容的 `origin_set`

**操作**：
1.  创建另一个同样基于 `origin_group_id=2` 的 `origin_set`（`id=13`）。
2.  将 `website_id=41` 切换到 `origin_set_id=13`。

```shell
# 创建 origin_set_id=13 (已执行)

# 切换绑定
curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-sets/bind-website" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{"websiteId":41,"originSetId":13}'
```

**返回**：

```json
{"code":0,"message":"success","data":{"item":{"originSetId":13,"skipReason":"content unchanged","taskCreated":false,"taskId":null,"websiteId":41}}}
```

**验证**：`taskCreated` 为 `false`，`skipReason` 为 `content unchanged`。两次绑定的 `contentHash` 经 `diff` 和 `sha256sum` 验证完全一致，说明幂等性修复成功。

```
# sha256sum /opt/go_cmdb/var/debug/hashpayload_41_*.json
9ebb7393dfd3067278995f234e6d348816002eaa9d1260b29504c9161f96552e  /opt/go_cmdb/var/debug/hashpayload_41_12.json
9ebb7393dfd3067278995f234e6d348816002eaa9d1260b29504c9161f96552e  /opt/go_cmdb/var/debug/hashpayload_41_13.json
```

## 四、结论

`origin-sets/create` 接口的数据填充缺陷和 `bind-website` 接口的幂等性缺陷均已成功修复。所有验收用例均已通过，系统行为符合预期。
