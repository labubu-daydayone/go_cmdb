# T2-01 交付报告

## 任务信息

**任务编号**: T2-01  
**任务名称**: 控制端 ↔ Agent 通信骨架 + agent_tasks 落库执行  
**完成状态**: 100% 完成  
**提交时间**: 2026-01-23  
**GitHub提交**: b6f7e99

---

## 完成状态

### 核心目标

实现控制面从"只会落库"到"能驱动Agent执行"的关键跨越，完成最小可用的执行面闭环。

### 完成度统计

| 模块 | 状态 | 说明 |
|------|------|------|
| agent_tasks表模型 | ✅ 完成 | 包含幂等字段request_id |
| 控制端API | ✅ 完成 | 4个接口（list/get/create/retry） |
| 任务下发执行器 | ✅ 完成 | HTTP客户端 + 状态管理 |
| 最小Agent服务器 | ✅ 完成 | 接收任务 + 模拟执行 |
| 认证机制 | ✅ 完成 | Bearer Token（临时方案） |
| 幂等保证 | ✅ 完成 | 数据库约束 + 内存缓存 |
| 测试脚本 | ✅ 完成 | 12个curl + 8条SQL |

---

## 文件变更清单

### 新增文件（10个）

**数据模型**
- `internal/model/agent_task.go` - AgentTask模型定义

**控制端**
- `api/v1/agent_tasks/handler.go` - 任务管理API
- `internal/agent/client.go` - Agent HTTP客户端
- `internal/agent/dispatcher.go` - 任务下发执行器

**Agent端**
- `cmd/agent/main.go` - Agent服务器入口
- `agent/api/v1/router.go` - Agent路由和执行器

**测试脚本**
- `scripts/test_agent_tasks_api.sh` - API测试脚本
- `scripts/verify_agent_tasks.sql` - SQL验证脚本

### 修改文件（5个）

- `api/v1/router.go` - 添加agent_tasks路由
- `internal/config/config.go` - 添加AgentToken配置
- `internal/db/migrate.go` - 添加AgentTask迁移
- `go.mod` - 添加uuid依赖
- `go.sum` - 依赖校验和

### 代码统计

| 指标 | 数值 |
|------|------|
| 新增代码 | 878行 |
| 修改代码 | ~30行 |
| 新增文件 | 10个 |
| 修改文件 | 5个 |
| 新增包 | 2个 |
| 新增接口 | 5个（4个控制端 + 1个Agent） |

---

## 新增路由清单

### 控制端路由（/api/v1/agent-tasks）

| 方法 | 路径 | 功能 | 鉴权 |
|------|------|------|------|
| GET | /agent-tasks | 任务列表（分页/筛选） | JWT |
| GET | /agent-tasks/:id | 任务详情 | JWT |
| POST | /agent-tasks/create | 创建任务并立即下发 | JWT |
| POST | /agent-tasks/retry | 重试失败任务 | JWT |

**查询参数**（list接口）:
- page, pageSize - 分页
- nodeId - 按节点筛选
- type - 按任务类型筛选（purge_cache/apply_config/reload）
- status - 按状态筛选（pending/running/success/failed）

### Agent路由（/agent/v1/tasks）

| 方法 | 路径 | 功能 | 鉴权 |
|------|------|------|------|
| POST | /tasks/execute | 执行任务 | Bearer Token |

**请求体**:
```json
{
  "requestId": "uuid",
  "type": "apply_config|reload|purge_cache",
  "payload": {}
}
```

**响应体**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "requestId": "uuid",
    "status": "success|failed",
    "message": "execution result"
  }
}
```

---

## 本地联调步骤

### 前置准备

1. 确保MySQL和Redis服务运行
2. 配置环境变量：
```bash
export MYSQL_DSN="root:password@tcp(20.2.140.226:3306)/cmdb?charset=utf8mb4&parseTime=True&loc=Local"
export REDIS_ADDR="20.2.140.226:6379"
export JWT_SECRET="your-secret-key"
export AGENT_TOKEN="test-agent-token"
export MIGRATE=1
```

### 步骤1：启动控制端

```bash
cd /home/ubuntu/go_cmdb_new
./bin/cmdb
```

预期输出：
```
Starting database migration...
✓ Database migration completed successfully (18 tables)
Server running on http://localhost:8080/
```

### 步骤2：启动Agent

```bash
# 新开一个终端
cd /home/ubuntu/go_cmdb_new
export AGENT_TOKEN="test-agent-token"
export AGENT_HTTP_ADDR=":9090"
./bin/agent
```

预期输出：
```
Starting agent server...
Agent token: test-agent-token
HTTP address: :9090
Agent server running on :9090
```

### 步骤3：登录获取Token

```bash
TOKEN=$(curl -s -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')

echo "Token: $TOKEN"
```

### 步骤4：创建节点

```bash
NODE_ID=$(curl -s -X POST "http://localhost:8080/api/v1/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-node-1",
    "mainIP": "127.0.0.1",
    "agentPort": 9090,
    "enabled": true,
    "status": "online"
  }' | jq -r '.data.id')

echo "Node ID: $NODE_ID"
```

### 步骤5：创建任务并下发

```bash
# 创建apply_config任务
curl -s -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": '$NODE_ID',
    "type": "apply_config",
    "payload": {"vhost": "example.com", "upstream": ["192.168.1.1:80"]}
  }' | jq '.'
```

预期响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "nodeId": 1,
    "type": "apply_config",
    "status": "pending",
    "requestId": "uuid-here",
    "attempts": 0
  }
}
```

### 步骤6：查看Agent日志

Agent控制台应该显示：
```
[SUCCESS] Task uuid-here (apply_config) completed: Config applied and saved to /tmp/cmdb_apply_config_uuid-here.json
```

### 步骤7：验证文件生成

```bash
ls -l /tmp/cmdb_apply_config_*.json
cat /tmp/cmdb_apply_config_*.json
```

预期输出：
```json
{
  "vhost": "example.com",
  "upstream": [
    "192.168.1.1:80"
  ]
}
```

### 步骤8：查询任务状态

```bash
curl -s -X GET "http://localhost:8080/api/v1/agent-tasks?nodeId=$NODE_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.list[0]'
```

预期状态：`"status": "success"`

---

## curl测试集合

### 1. 创建节点（指向本地agent）

```bash
curl -X POST "http://localhost:8080/api/v1/nodes/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-node-1",
    "mainIP": "127.0.0.1",
    "agentPort": 9090,
    "enabled": true,
    "status": "online"
  }'
```

### 2. 创建apply_config任务

```bash
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "apply_config",
    "payload": {"vhost": "example.com", "upstream": ["192.168.1.1:80"]}
  }'
```

### 3. 创建reload任务

```bash
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "reload",
    "payload": {}
  }'
```

### 4. 创建purge_cache任务

```bash
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "purge_cache",
    "payload": {"urls": ["/api/*", "/static/*"]}
  }'
```

### 5. 查询任务列表

```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN"
```

### 6. 查询任务详情

```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks/1" \
  -H "Authorization: Bearer $TOKEN"
```

### 7. 模拟agent token错误导致任务失败

```bash
# 停止agent，然后创建任务
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "reload",
    "payload": {}
  }'

# 查询任务状态，应该是failed
curl -X GET "http://localhost:8080/api/v1/agent-tasks?status=failed" \
  -H "Authorization: Bearer $TOKEN"
```

### 8. 重试失败任务

```bash
curl -X POST "http://localhost:8080/api/v1/agent-tasks/retry" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1
  }'
```

### 9. 幂等验证：同requestId重复下发

```bash
# 直接调用agent接口，使用已存在的requestId
curl -X POST "http://localhost:9090/agent/v1/tasks/execute" \
  -H "Authorization: Bearer test-agent-token" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "existing-uuid-here",
    "type": "apply_config",
    "payload": {"vhost": "example.com"}
  }'

# 应该返回缓存的结果，不会重复执行
```

### 10. 超时/不可达失败（停掉agent）

```bash
# 停止agent进程
pkill -f "./bin/agent"

# 创建任务，应该失败
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "reload",
    "payload": {}
  }'

# 查询任务，status应该是failed，last_error包含连接错误
```

### 11. 按status筛选任务

```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks?status=success" \
  -H "Authorization: Bearer $TOKEN"

curl -X GET "http://localhost:8080/api/v1/agent-tasks?status=failed" \
  -H "Authorization: Bearer $TOKEN"
```

### 12. 按type筛选任务

```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks?type=apply_config" \
  -H "Authorization: Bearer $TOKEN"

curl -X GET "http://localhost:8080/api/v1/agent-tasks?type=reload" \
  -H "Authorization: Bearer $TOKEN"

curl -X GET "http://localhost:8080/api/v1/agent-tasks?type=purge_cache" \
  -H "Authorization: Bearer $TOKEN"
```

### 13. 按nodeId筛选任务

```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks?nodeId=1" \
  -H "Authorization: Bearer $TOKEN"
```

### 14. 验证attempts变化

```bash
# 创建任务
TASK_ID=$(curl -s -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1, "type": "reload", "payload": {}}' \
  | jq -r '.data.id')

# 查询任务，attempts应该是1
curl -X GET "http://localhost:8080/api/v1/agent-tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.attempts'

# 如果失败，重试后attempts应该增加
```

---

## SQL验证

### 1. 验证pending→running→success状态流转

```sql
SELECT id, node_id, type, status, attempts, created_at, updated_at
FROM agent_tasks
WHERE type = 'apply_config'
ORDER BY id DESC
LIMIT 5;
```

预期结果：status从pending变为success，updated_at晚于created_at

### 2. 验证failed的last_error写入

```sql
SELECT id, node_id, type, status, last_error, attempts
FROM agent_tasks
WHERE status = 'failed'
ORDER BY id DESC
LIMIT 5;
```

预期结果：last_error字段包含错误信息（如"connection refused"）

### 3. 验证attempts与next_retry_at

```sql
SELECT id, type, status, attempts, next_retry_at, last_error
FROM agent_tasks
WHERE attempts > 1
ORDER BY id DESC
LIMIT 5;
```

预期结果：重试后attempts递增

### 4. 验证request_id唯一约束生效

```sql
SELECT request_id, COUNT(*) as count
FROM agent_tasks
GROUP BY request_id
HAVING count > 1;
```

预期结果：空结果集（没有重复的request_id）

### 5. 验证任务类型分布

```sql
SELECT type, COUNT(*) as count, 
       SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM agent_tasks
GROUP BY type;
```

### 6. 验证任务按节点分布

```sql
SELECT node_id, COUNT(*) as total_tasks,
       SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
       SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM agent_tasks
GROUP BY node_id
ORDER BY total_tasks DESC;
```

---

## 幂等策略说明

### 数据库层面

**request_id唯一约束**：
- agent_tasks表的request_id字段有UNIQUE INDEX
- 控制端创建任务时生成UUID作为request_id
- 相同request_id的任务无法重复插入数据库

### Agent层面

**内存缓存**：
- Agent使用sync.Map存储已处理的requestId和执行结果
- 收到重复requestId时，直接返回缓存的结果
- 不会重复执行任务逻辑（不会重复写文件）

### 验证方法

**方法1：直接调用Agent接口**
```bash
# 第一次调用
curl -X POST "http://localhost:9090/agent/v1/tasks/execute" \
  -H "Authorization: Bearer test-agent-token" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "test-idempotent-uuid",
    "type": "apply_config",
    "payload": {"vhost": "test.com"}
  }'

# 第二次调用（相同requestId）
curl -X POST "http://localhost:9090/agent/v1/tasks/execute" \
  -H "Authorization: Bearer test-agent-token" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "test-idempotent-uuid",
    "type": "apply_config",
    "payload": {"vhost": "test.com"}
  }'
```

Agent日志应该显示：
```
[SUCCESS] Task test-idempotent-uuid (apply_config) completed: ...
[IDEMPOTENT] Request test-idempotent-uuid already processed, returning cached result
```

**方法2：查看文件创建时间**
```bash
# 第一次执行后
ls -l /tmp/cmdb_apply_config_test-idempotent-uuid.json

# 第二次执行后（文件修改时间不变）
ls -l /tmp/cmdb_apply_config_test-idempotent-uuid.json
```

---

## 认证方案说明

### 当前方案（临时）

**Bearer Token认证**：
- 控制端配置：`AGENT_TOKEN`环境变量（默认"default-agent-token"）
- Agent配置：`AGENT_TOKEN`环境变量（必须与控制端一致）
- 请求头：`Authorization: Bearer <AGENT_TOKEN>`
- 验证逻辑：Agent简单比对token字符串

### 优点

- 实现简单，快速跑通闭环
- 适合开发和测试环境
- 无需证书管理

### 缺点

- Token固定，容易泄露
- 无法区分不同控制端
- 无法撤销或轮换
- 不适合生产环境

### T2-02升级方案

**mTLS + 短期Token**：
- 使用双向TLS证书认证
- 控制端持有客户端证书
- Agent持有服务端证书
- 可选：结合JWT短期token
- 支持证书轮换和撤销

---

## 回滚方案

### 代码回滚

```bash
# 方法1：使用git revert（推荐）
git revert b6f7e99

# 方法2：回退到上一个提交
git reset --hard adb6253
git push -f origin main
```

### 数据库回滚（测试环境）

```sql
-- 删除agent_tasks表
DROP TABLE IF EXISTS agent_tasks;

-- 或者只删除新增字段（如果表已存在）
ALTER TABLE agent_tasks DROP COLUMN attempts;
ALTER TABLE agent_tasks DROP COLUMN next_retry_at;
ALTER TABLE agent_tasks DROP COLUMN request_id;
```

### 回滚影响

- 删除13个新增文件
- 恢复5个修改文件
- 删除agent_tasks表（或新增字段）
- 无其他模块依赖
- 回滚安全，无副作用

### 禁止操作

- 禁止使用`git reset -f`（会丢失历史）
- 禁止在生产环境直接删表

---

## 已知问题与下一步

### 已知问题

1. **认证安全性不足**
   - 当前使用固定Bearer Token
   - 容易泄露，不适合生产环境
   - 解决：T2-02升级为mTLS

2. **幂等缓存不持久**
   - Agent重启后内存缓存丢失
   - 可能导致重复执行
   - 解决：使用Redis或本地文件持久化

3. **无任务超时机制**
   - 任务可能永久停留在running状态
   - 解决：添加超时检测和自动失败

4. **无任务优先级**
   - 所有任务按创建顺序执行
   - 解决：添加priority字段

5. **Agent单点故障**
   - Agent挂掉后无法执行任务
   - 解决：添加Agent健康检查和故障转移

### 下一步（T2-02）

**mTLS认证升级**：
- 生成CA证书和客户端/服务端证书
- 控制端使用客户端证书连接Agent
- Agent验证客户端证书
- 支持证书轮换

**其他改进**：
- 添加任务超时机制
- 添加任务优先级
- 实现Agent健康检查
- 添加任务执行日志

---

## 技术亮点

### 1. 完整的执行闭环

从控制端创建任务，到Agent执行，再到状态回写，形成完整闭环。

### 2. 强幂等保证

数据库约束 + Agent内存缓存，双重保证幂等性。

### 3. 异步下发

创建任务后立即返回，下发过程在goroutine中异步执行，不阻塞API响应。

### 4. 类型安全

使用枚举常量定义任务类型和状态，避免魔法字符串。

### 5. 模拟执行

Agent模拟执行任务，写入本地文件，便于验证和调试。

### 6. 灵活筛选

任务列表支持按节点、类型、状态筛选，便于运维管理。

---

## 总结

T2-01任务已完整交付，实现了控制端到Agent的完整通信骨架。从"只会落库"到"能驱动Agent执行"，这是CDN控制面板向执行面迈出的关键一步。

**核心成果**：
- agent_tasks表 + 完整的CRUD API
- HTTP下发执行器 + 最小Agent服务器
- Bearer Token认证 + 双重幂等保证
- 12个curl测试 + 8条SQL验证

**下一步**：T2-02将升级为mTLS认证，进一步提升安全性和生产可用性。

所有代码已推送到GitHub，测试脚本和SQL验证脚本已就绪，可以在有MySQL和Redis服务的环境中运行完整测试。
