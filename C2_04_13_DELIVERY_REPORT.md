# C2-04-13-CP-01: Agent 任务拉取与状态回写接口实现报告

## 1. 任务目标

在控制端新增 Agent 任务拉取和状态回写接口，实现 Agent 执行闭环。

- **任务拉取**: `GET /api/v1/agent/tasks/pull`
- **状态回写**: `POST /api/v1/agent/tasks/update-status`

## 2. 实现方案

### 2.1. 接口设计

#### 2.1.1. 认证机制

- 优先使用 `X-Node-Id` HTTP Header 进行节点认证（用于测试）。
- 保留 mTLS 客户端证书认证的扩展入口。
- 若无有效认证信息，返回 `code=2001`。

#### 2.1.2. 状态映射

为保持数据库结构不变，在接口层面进行状态映射：

| 数据库 (`agent_tasks.status`) | API 接口 | 说明 |
| :--- | :--- | :--- |
| `pending` | `pending` | 任务待领取 |
| `running` | `running` | 任务领取成功，执行中 |
| `success` | `succeeded` | 任务执行成功 |
| `failed` | `failed` | 任务执行失败 |

### 2.2. `GET /api/v1/agent/tasks/pull` 实现

1.  **认证**: 从 `X-Node-Id` Header 获取 `nodeId`。
2.  **查询**: 查询 `agent_tasks` 表中该 `nodeId` 的 `pending` 或 `retrying` 状态的任务 ID 列表。
3.  **原子更新**: 使用 `UPDATE ... WHERE id IN (...) AND status IN (...)` 将任务状态从 `pending`/`retrying` 更新为 `running`，确保任务领取的原子性。
4.  **返回任务**: 查询并返回成功更新为 `running` 状态的任务列表，并将 `status` 字段从 `success` 映射为 `succeeded`。

### 2.3. `POST /api/v1/agent/tasks/update-status` 实现

1.  **认证**: 从 `X-Node-Id` Header 获取 `nodeId`。
2.  **参数校验**: 解析请求体，校验 `taskId` 和 `status` (`succeeded` 或 `failed`)。
3.  **任务校验**: 查询 `taskId` 对应的任务，验证该任务是否属于当前 `nodeId`，且状态为 `running`。
4.  **状态更新**: 将 API 的 `succeeded` 状态映射为数据库的 `success`，并更新任务状态。若为 `failed`，则同时更新 `last_error` 字段。

## 3. 验收过程

### 3.1. 准备工作

- 编译并重启服务。
- 在数据库中手动创建一条 `pending` 状态的 `agent_task` 用于测试。

### 3.2. 测试用例

| 步骤 | 操作 | 预期结果 |
| :--- | :--- | :--- |
| 1 | 拉取任务 (`/pull`) | 成功返回任务，状态为 `running` |
| 2 | 重复拉取 (`/pull`) | 返回空列表，任务已被领取 |
| 3 | 回写成功 (`/update-status`) | 返回成功，数据库状态更新为 `success` |
| 4 | 重复回写 | 返回错误，任务状态非 `running` |
| 5 | 创建新任务并拉取 | 成功返回新任务 |
| 6 | 回写失败 (`/update-status`) | 返回成功，数据库状态更新为 `failed`，并记录 `last_error` |

### 3.3. 关键命令与返回

#### 拉取任务

```bash
curl -s -X GET "http://20.2.140.226:8080/api/v1/agent/tasks/pull" -H "X-Node-Id: 12"
```

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "id": 3,
                "status": "running",
                ...
            }
        ]
    }
}
```

#### 回写成功

```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/agent/tasks/update-status" \
  -H "Content-Type: application/json" \
  -H "X-Node-Id: 12" \
  -d '{"taskId": 3, "status": "succeeded"}'
```

```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

#### 回写失败

```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/agent/tasks/update-status" \
  -H "Content-Type: application/json" \
  -H "X-Node-Id: 12" \
  -d '{"taskId": 4, "status": "failed", "errorMessage": "test error message"}'
```

```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 4. 结论

Agent 任务拉取和状态回写接口已成功实现，并通过所有验收测试用例。系统现已具备 Agent 执行任务的完整闭环能力。
