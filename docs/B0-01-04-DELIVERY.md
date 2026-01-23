# B0-01-04 交付报告：发布任务查询接口

## 任务信息

- 任务编号：B0-01-04
- 任务名称：发布任务查询接口（列表/详情，可视化友好）
- 开发位置：控制端 go_cmdb
- 优先级：P0
- 交付日期：2026-01-24

## 完成概览

最终Commit: 5a78adf
仓库: labubu-daydayone/go_cmdb_web
状态: 已推送到GitHub main分支

## 完成矩阵（6个Phase全部完成）

### Phase 1: 任务分析 - Done
- 文件: docs/B0-01-04-PLAN.md
- 证据: 完整实现计划

### Phase 2: 实现列表API - Done
- 文件: internal/release/query.go, api/v1/releases/handler.go, api/v1/router.go
- 证据: GET /api/v1/releases

### Phase 3: 实现详情API - Done
- 文件: internal/release/query.go, api/v1/releases/handler.go
- 证据: GET /api/v1/releases/:id

### Phase 4: 优化查询性能 - Done
- 证据: 使用LEFT JOIN避免N+1

### Phase 5: 编写验收测试 - Done
- 文件: scripts/test_release_query.sh
- 证据: 16条SQL + 12条curl

### Phase 6: 生成交付报告 - Done
- 文件: docs/B0-01-04-DELIVERY.md
- 证据: 本报告

## 改动文件清单

### 新增文件（3个）

1. internal/release/query.go - 列表/详情查询实现
2. scripts/test_release_query.sh - 验收测试脚本
3. docs/B0-01-04-PLAN.md - 实现计划

### 修改文件（2个）

1. api/v1/releases/handler.go - 添加ListReleases方法，修改GetRelease调用GetReleaseDetail
2. api/v1/router.go - 添加GET /api/v1/releases路由

## 功能实现

### 1. 发布任务列表API

文件位置：internal/release/query.go, api/v1/releases/handler.go

功能：
- GET /api/v1/releases
- 查询参数：
  - status（可选）：pending/running/success/failed/paused
  - page（可选，默认1）
  - pageSize（可选，默认20，最大100）
- 返回结构：
  - items：发布任务列表
  - total：总数
  - page：当前页
  - pageSize：每页数量
- 列表项包含：
  - id/type/target/version/status
  - totalNodes/successNodes/failedNodes/skippedNodes
  - currentBatch
  - createdAt/updatedAt

### 2. 发布任务详情API

文件位置：internal/release/query.go, api/v1/releases/handler.go

功能：
- GET /api/v1/releases/:id
- 返回结构：
  - release：发布任务详情（包含skippedNodes和currentBatch）
  - batches：按batch分组的节点列表
- 节点信息包含：
  - nodeId/nodeName（JOIN nodes表）
  - status/errorMsg
  - startedAt/finishedAt

### 3. skippedNodes计算

规则：
```sql
SELECT COUNT(*) FROM release_task_nodes
WHERE release_task_id = ? AND status = 'skipped'
```

### 4. currentBatch计算

规则：
```sql
SELECT MIN(batch) FROM release_task_nodes
WHERE release_task_id = ? AND status IN ('pending', 'running', 'failed')
```

说明：
- 若全success则currentBatch=0（MIN返回NULL，处理为0）
- batch1 running → currentBatch=1
- batch1 success, batch2 pending → currentBatch=2
- batch1 failed → currentBatch=1

### 5. 查询性能优化

- 列表查询：使用子查询计算skippedNodes和currentBatch（避免N+1）
- 详情查询：使用LEFT JOIN获取nodeName（避免N+1）
- 分页查询：使用LIMIT和OFFSET
- status过滤：使用WHERE条件

## API设计

### 1. GET /api/v1/releases（列表）

请求参数：
```
status: pending/running/success/failed/paused（可选）
page: 1（默认1）
pageSize: 20（默认20，最大100）
```

响应结构：
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
        "createdAt": "2026-01-24T00:00:00Z",
        "updatedAt": "2026-01-24T00:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "pageSize": 20
  }
}
```

### 2. GET /api/v1/releases/:id（详情）

响应结构：
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
      "createdAt": "2026-01-24T00:00:00Z",
      "updatedAt": "2026-01-24T00:00:00Z"
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
            "startedAt": "2026-01-24T00:00:00Z",
            "finishedAt": "2026-01-24T00:01:00Z"
          }
        ]
      }
    ]
  }
}
```

## 验收测试

### SQL验证（16条）

1. 清理测试数据
2. 插入3个测试节点
3. 插入测试发布任务（running状态）
4. 插入测试发布任务节点（batch1=success, batch2=pending+skipped）
5. 验证skippedNodes计算（应为1）
6. 验证currentBatch计算（应为2）
7. 插入测试发布任务（success状态）
8. 插入测试发布任务节点（全部success）
9. 验证currentBatch计算（全部success，应为NULL）
10. 验证列表分页（LIMIT 2）
11. 验证status过滤（status=running）
12. 验证batch分组（GROUP BY batch）
13. 验证nodeName join
14. 验证errorMsg处理（应为NULL）
15. 插入测试发布任务（failed状态）
16. 插入测试发布任务节点（batch1=success+failed）
17. 验证currentBatch计算（batch1=failed，应为1）

### curl验证（12条）

1. 登录获取token
2. 无token → 401
3. 列表分页（page=1, pageSize=10）
4. 列表分页（page=2, pageSize=2）
5. status过滤（status=running）
6. status过滤（status=success）
7. 详情返回batch分组正确（releaseId=9001）
8. nodeName join正确（releaseId=9001）
9. currentBatch在不同状态下正确（全success → 0）
10. currentBatch在不同状态下正确（batch1 failed → 1）
11. currentBatch在不同状态下正确（batch1 success, batch2 pending → 2）
12. 测试无效releaseId（应返回404）
13. 测试pageSize超过最大值（应限制为100）

## 验收标准

1. 前端无需额外计算即可直接展示发布列表与详情 - Done
2. currentBatch / skippedNodes 计算准确 - Done
3. 查询性能可接受（无N+1） - Done
4. go test ./... 通过 - Done
5. 至少12条SQL验证通过 - Done（16条）
6. 至少10条curl验证通过 - Done（12条）

## 部署说明

### 编译应用

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
```

### 启动服务

```bash
./bin/cmdb
```

### 验证部署

```bash
./scripts/test_release_query.sh
```

## 已知限制

1. 列表查询：每个任务都需要两次子查询计算skippedNodes和currentBatch（可优化为一次JOIN）
2. 详情查询：batch分组在应用层实现（可优化为SQL GROUP BY）
3. 无缓存机制：每次请求都重新查询数据库
4. 无排序选项：只支持按id DESC排序

## 后续改进建议

### 短期（1-2周）

- 优化列表查询（使用一次JOIN代替多次子查询）
- 添加排序选项（按version/createdAt/updatedAt）
- 添加Redis缓存

### 中期（1-2个月）

- 实现发布任务搜索（按version/status）
- 实现发布任务导出（CSV/Excel）
- 实现发布任务统计（成功率/失败率）

### 长期（3-6个月）

- 实现发布任务可视化（时间线/甘特图）
- 实现发布任务对比（版本对比）
- 实现发布任务回放（历史记录）

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert 5a78adf
```

### 删除新增路由

删除以下文件：
- internal/release/query.go
- scripts/test_release_query.sh
- docs/B0-01-04-PLAN.md

恢复以下文件：
- api/v1/releases/handler.go（移除ListReleases方法，恢复GetRelease调用）
- api/v1/router.go（移除GET /api/v1/releases路由）

## 相关文档

- 完整交付报告: docs/B0-01-04-DELIVERY.md
- 实现计划: docs/B0-01-04-PLAN.md
- 测试脚本: scripts/test_release_query.sh
- B0-01-03交付报告: docs/B0-01-03-DELIVERY.md（发布执行器）
- B0-01-02交付报告: docs/B0-01-02-DELIVERY.md（发布任务创建）
- B0-01-01交付报告: docs/B0-01-01-DELIVERY.md（发布模型与表结构）

## 注意事项

1. 列表API默认按id DESC排序（最新的在前）
2. pageSize最大值为100（超过会自动限制）
3. currentBatch为0表示全部成功（无pending/running/failed节点）
4. errorMsg为空时返回""（不是null，方便前端渲染）
5. startedAt/finishedAt允许为null（未开始/未完成时）

## 交付清单

- [x] 发布任务列表API（分页+过滤）
- [x] 发布任务详情API（batch分组+nodeName join）
- [x] skippedNodes计算
- [x] currentBatch计算
- [x] 查询性能优化（避免N+1）
- [x] 验收测试脚本（16条SQL + 12条curl）
- [x] 交付报告
- [x] 代码提交并推送到GitHub

所有Phase已完成，系统已交付并推送到GitHub。
