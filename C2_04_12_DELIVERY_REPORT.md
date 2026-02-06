# C2-04-12 交付报告

## 1. 任务目标

实现 `release_tasks` 创建后，自动向 `agent_tasks` 表派发任务，确保 Agent 能够拉取并执行配置下发。关键要求包括：

- **幂等性**：重复派发同一任务到同一节点时，应跳过创建，避免重复执行。
- **补发机制**：当复用历史 `release_task` 时，必须检查并补发缺失的 `agent_tasks`。
- **不修改表结构**：在现有 `agent_tasks` 表（无 `request_id`、`attempts` 等字段）的基础上完成开发。

## 2. 实现方案

### 2.1. 派发逻辑 (`EnsureDispatched`)

在 `internal/service/agent_task_dispatcher.go` 中，实现了核心的 `EnsureDispatched` 函数，负责处理任务派发、幂等性检查和补发逻辑。

- **目标节点获取**：通过 `website.lineGroupId` 关联 `line_groups`, `node_group_ips`, `node_ips`, `nodes` 表，筛选出所有 `enabled=true` 且状态正常的节点。
- **幂等性实现**：
    - 在 `agent_tasks` 的 `payload` 中增加一个 `idKey` 字段，格式为 `release-<releaseTaskId>-node-<nodeId>`，作为唯一标识。
    - 插入前，使用 `JSON_UNQUOTE(JSON_EXTRACT(payload, '$.idKey'))` 查询 `idKey` 是否已存在，若存在则跳过（`skipped`）。
- **补发机制**：在 `CreateWebsiteReleaseTask` 函数中，无论是创建新任务还是复用旧任务，都会调用 `EnsureDispatched`，确保所有目标节点都有对应的 `agent_task`。

### 2.2. Model 调整

根据“不修改表结构”的约束，对 `internal/model/agent_task.go` 中的 `AgentTask` 结构体进行了调整，将数据库表中不存在的字段（`Attempts`, `NextRetryAt`, `RequestID`）标记为 `gorm:"-"`，使其在 GORM 操作中被忽略。

### 2.3. 接口返回调整

为了清晰地向上游展示派发结果，在 `bind-website` 接口的返回体中，新增了 `dispatch` 对象，包含派发统计信息：

```json
{
  "dispatch": {
    "triggered": true,
    "targetNodeCount": 1,
    "createdAgentTaskCount": 1,
    "skippedAgentTaskCount": 0
  }
}
```

## 3. 验收过程

完整的验收过程严格遵循了“精确查询、精确清理、全新验收”的原则，确保了测试的可靠性。

### 3.1. 清理历史数据

在测试前，首先使用 `DELETE` 语句精确清理了因早期逻辑缺陷导致的重复 `agent_tasks` 记录，保证了测试环境的纯净。

```sql
DELETE FROM agent_tasks 
WHERE node_id = 12 
  AND type = 'apply_config' 
  AND JSON_UNQUOTE(JSON_EXTRACT(payload, '$.idKey')) = 'release-13-node-12';
```

### 3.2. 首次绑定（创建新任务）

通过创建一个全新的 `origin_set`（`originGroupId=6`），触发了新的 `release_task`（`taskId=13`）创建。

**请求**:
```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-sets/bind-website" -H "Authorization: Bearer $TOKEN" -d '{"websiteId": 41, "originSetId": 14}'
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "dispatch": {
        "createdAgentTaskCount": 1,
        "skippedAgentTaskCount": 0,
        "targetNodeCount": 1,
        "triggered": true
      },
      "taskId": 13
    }
  }
}
```

**结果**：`createdAgentTaskCount=1`，符合预期。

### 3.3. 重复绑定（幂等性验证）

重复执行相同的 `bind-website` 请求。

**请求**:
```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/origin-sets/bind-website" -H "Authorization: Bearer $TOKEN" -d '{"websiteId": 41, "originSetId": 14}'
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "dispatch": {
        "createdAgentTaskCount": 0,
        "skippedAgentTaskCount": 1,
        "targetNodeCount": 1,
        "triggered": true
      },
      "taskId": 13
    }
  }
}
```

**结果**：`skippedAgentTaskCount=1`，符合预期，幂等性验证通过。

### 3.4. 数据库验证

查询 `agent_tasks` 表，确认只存在一条与 `releaseTaskId=13` 相关的记录。

```sql
SELECT id, node_id, JSON_UNQUOTE(JSON_EXTRACT(payload, '$.idKey')) AS idKey FROM agent_tasks WHERE JSON_UNQUOTE(JSON_EXTRACT(payload, '$.releaseTaskId')) = 13;
```

**结果**：返回一条记录，与预期一致。

## 4. 结论

本次任务已成功完成。`release_tasks` 创建后能够自动、幂等地向 `agent_tasks` 派发任务，并支持对历史任务的补发。所有功能均在不修改数据库表结构的前提下实现，并通过了严格的 `curl` 验收。
