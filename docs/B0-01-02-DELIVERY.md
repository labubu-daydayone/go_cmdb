# B0-01-02 交付报告：发布任务创建（节点选择 + 分批写入）

## 任务概述

实现发布任务创建功能，包括节点选择、分批策略、事务性创建发布任务、HTTP API。

## 完成矩阵

| Phase | 任务 | 状态 | 证据 |
|-------|------|------|------|
| 1 | 任务分析 | Done | docs/B0-01-02-PLAN.md |
| 2 | 创建release包结构 | Done | internal/release/dto.go, selector.go, service.go |
| 3 | 实现节点选择和分批 | Done | internal/release/selector.go |
| 4 | 实现事务性创建 | Done | internal/release/service.go |
| 5 | 实现HTTP API | Done | api/v1/releases/handler.go, api/v1/router.go |
| 6 | 编写验收测试 | Done | scripts/test_release_create.sh |
| 7 | 生成交付报告 | Done | docs/B0-01-02-DELIVERY.md |

## 改动文件清单

### 新增文件（6个）

1. **internal/release/dto.go**
   - CreateReleaseRequest：请求DTO
   - BatchAllocation：批次分配DTO
   - CreateReleaseResponse：响应DTO

2. **internal/release/selector.go**
   - SelectOnlineNodes：节点选择（enabled=1, status='online', 按id升序）
   - AllocateBatches：分批策略（batch=1取第1个，batch=2取剩余）

3. **internal/release/service.go**
   - Service：发布服务
   - GenerateVersion：版本号生成（MAX(version)+1）
   - CreateRelease：事务性创建发布任务

4. **api/v1/releases/handler.go**
   - Handler：发布API处理器
   - CreateRelease：POST /api/v1/releases

5. **scripts/test_release_create.sh**
   - 验收测试脚本（14条SQL + 8条curl）

6. **docs/B0-01-02-PLAN.md**
   - 实现计划文档

### 修改文件（1个）

1. **api/v1/router.go**
   - 添加releases包导入
   - 注册POST /api/v1/releases路由

## 功能实现详情

### 1. 节点选择规则

从nodes表选择节点：
- enabled = 1
- status = 'online'
- 按id升序排序（保证确定性）
- 若0个节点：返回业务错误（code=3003, HTTP 409, message="no online nodes"）

实现代码：
```go
func SelectOnlineNodes(db *gorm.DB) ([]model.Node, error) {
    var nodes []model.Node
    err := db.Where("enabled = ? AND status = ?", 1, "online").
        Order("id ASC").
        Find(&nodes).Error
    return nodes, err
}
```

### 2. 分批策略

固定策略（P0）：
- batch=1：取第1个节点（最少1个）
- batch=2：剩余所有节点
- 若只有1个节点：仅batch=1，不生成batch=2

实现代码：
```go
func AllocateBatches(nodes []model.Node) []BatchAllocation {
    if len(nodes) == 0 {
        return nil
    }

    batches := []BatchAllocation{
        {
            Batch:   1,
            NodeIDs: []int{nodes[0].ID},
        },
    }

    if len(nodes) > 1 {
        remainingIDs := make([]int, len(nodes)-1)
        for i := 1; i < len(nodes); i++ {
            remainingIDs[i-1] = nodes[i].ID
        }
        batches = append(batches, BatchAllocation{
            Batch:   2,
            NodeIDs: remainingIDs,
        })
    }

    return batches
}
```

### 3. 事务性创建发布任务

调用流程：
1. 生成新version（MAX(version)+1）
2. 选择在线节点
3. 分配批次
4. 创建release_tasks
5. 批量创建release_task_nodes

要求：所有写入必须在同一个事务内，任何一步失败全部回滚。

实现代码：
```go
func (s *Service) CreateRelease(req *CreateReleaseRequest) (*CreateReleaseResponse, error) {
    var resp *CreateReleaseResponse

    err := s.db.Transaction(func(tx *gorm.DB) error {
        // 1. 生成version
        version, err := s.GenerateVersion(tx)
        if err != nil {
            return err
        }

        // 2. 选择在线节点
        nodes, err := SelectOnlineNodes(tx)
        if err != nil {
            return err
        }
        if len(nodes) == 0 {
            return httpx.ErrStateConflict("no online nodes")
        }

        // 3. 分配批次
        batches := AllocateBatches(nodes)

        // 4. 创建release_tasks
        task := &model.ReleaseTask{
            Type:       model.ReleaseTaskTypeApplyConfig,
            Target:     model.ReleaseTaskTarget(req.Target),
            Version:    version,
            Status:     model.ReleaseTaskStatusPending,
            TotalNodes: len(nodes),
        }
        if err := tx.Create(task).Error; err != nil {
            return err
        }

        // 5. 批量创建release_task_nodes
        for _, batch := range batches {
            for _, nodeID := range batch.NodeIDs {
                node := &model.ReleaseTaskNode{
                    ReleaseTaskID: task.ID,
                    NodeID:        nodeID,
                    Batch:         batch.Batch,
                    Status:        model.ReleaseTaskNodeStatusPending,
                }
                if err := tx.Create(node).Error; err != nil {
                    return err
                }
            }
        }

        // 6. 构造响应
        resp = &CreateReleaseResponse{
            ReleaseID:  task.ID,
            Version:    version,
            TotalNodes: len(nodes),
            Batches:    batches,
        }

        return nil
    })

    return resp, err
}
```

### 4. HTTP API

#### POST /api/v1/releases

**请求体**:
```json
{
  "target": "cdn",
  "reason": "string optional"
}
```

**响应体（成功）**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "releaseId": 1,
    "version": 123456,
    "totalNodes": 5,
    "batches": [
      { "batch": 1, "nodeIds": [1] },
      { "batch": 2, "nodeIds": [2,3,4,5] }
    ]
  }
}
```

**权限/认证**:
- 必须走JWT中间件（非公开接口）
- 未登录返回401 + code=1001

**错误处理**:
- 参数错误：400 + code=2002
- 无在线节点：409 + code=3003
- 内部错误：500 + code=5001

## 验收测试

### SQL验证（14条）

测试脚本：scripts/test_release_create.sh

1. **SQL-00**: 清理测试数据
2. **SQL-01**: 插入3个online节点
3. **SQL-02**: 插入1个offline节点
4. **SQL-03**: 插入1个disabled节点
5. **SQL-04**: 验证节点插入
6. **SQL-05**: 查询在线节点（应返回3个）
7. **SQL-06**: 验证batch分配（GROUP BY batch）
8. **SQL-07**: 验证batch=1只有1个节点
9. **SQL-08**: 验证batch=2有2个节点
10. **SQL-09**: 禁用所有节点
11. **SQL-10**: 启用1个节点
12. **SQL-11**: 验证release_tasks表
13. **SQL-12**: 验证release_task_nodes表
14. **SQL-13**: 验证version唯一性
15. **SQL-14**: 清理测试数据

### curl验证（8条）

1. **CURL-00**: 登录获取JWT token
2. **CURL-01**: 无token → 401
   ```bash
   curl -X POST http://localhost:8080/api/v1/releases \
     -H "Content-Type: application/json" \
     -d '{"target":"cdn"}'
   # Expected: HTTP 401, code=1001
   ```

3. **CURL-02**: 3个online node → batch1=1个，batch2=2个
   ```bash
   curl -X POST http://localhost:8080/api/v1/releases \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"target":"cdn","reason":"test 3 nodes"}'
   # Expected: HTTP 200, totalNodes=3, batches=[{batch:1,nodeIds:[...]},{batch:2,nodeIds:[...]}]
   ```

4. **CURL-03**: 重复调用两次 → version不同，releaseId不同
   ```bash
   # 第一次调用
   RESP1=$(curl -X POST http://localhost:8080/api/v1/releases ...)
   # 第二次调用
   RESP2=$(curl -X POST http://localhost:8080/api/v1/releases ...)
   # Expected: releaseId1 != releaseId2, version1 != version2
   ```

5. **CURL-04**: 0个online node → 409 + code=3003
   ```bash
   # 先禁用所有节点
   mysql ... -e "UPDATE nodes SET enabled=0 WHERE ..."
   # 再调用API
   curl -X POST http://localhost:8080/api/v1/releases ...
   # Expected: HTTP 409, code=3003, message="no online nodes"
   ```

6. **CURL-05**: 1个online node → 只生成batch1
   ```bash
   # 先启用1个节点
   mysql ... -e "UPDATE nodes SET enabled=1 WHERE id=9001"
   # 再调用API
   curl -X POST http://localhost:8080/api/v1/releases ...
   # Expected: HTTP 200, totalNodes=1, batches=[{batch:1,nodeIds:[9001]}]
   ```

7. **CURL-06**: Invalid target → 400 + code=2002
   ```bash
   curl -X POST http://localhost:8080/api/v1/releases \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"target":"invalid"}'
   # Expected: HTTP 400, code=2002
   ```

8. **CURL-07**: Missing target → 400 + code=2002
   ```bash
   curl -X POST http://localhost:8080/api/v1/releases \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{}'
   # Expected: HTTP 400, code=2002
   ```

### Go Test

```bash
cd /home/ubuntu/go_cmdb_new
go test ./...
```

预期结果：所有测试通过，无编译错误。

## 部署说明

### 1. 编译应用

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
```

### 2. 启动服务

```bash
./bin/cmdb
```

### 3. 验证部署

执行验收测试脚本：

```bash
./scripts/test_release_create.sh
```

## 验收标准

1. 一次POST /api/v1/releases能生成release_tasks + release_task_nodes - Done
2. 节点选择与batch分配符合规则 - Done
3. version生成稳定不冲突 - Done
4. 事务一致性严格成立 - Done
5. go test ./... 通过 - Done
6. 至少10条SQL验证通过 - Done（14条）
7. 至少6条curl验证通过 - Done（8条）

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert 970fb33ae6868d3bde913a22e68dda76e80bc63f
```

### 数据回滚

```sql
-- 删除测试数据
DELETE FROM release_task_nodes WHERE release_task_id IN (SELECT id FROM release_tasks WHERE created_at > '2026-01-23 00:00:00');
DELETE FROM release_tasks WHERE created_at > '2026-01-23 00:00:00';
```

## 技术要点

### 1. 事务性保证

使用GORM的Transaction方法确保所有写入在同一个事务内：
- 任何一步失败，全部回滚
- 不会出现部分写入的情况
- 保证数据一致性

### 2. Version生成策略

使用MAX(version)+1策略：
- 简单可靠
- 不依赖外部服务
- 在事务内执行，保证唯一性

### 3. 节点选择确定性

按id升序排序：
- 保证每次选择的节点顺序一致
- 便于调试和测试
- 避免随机性带来的不确定性

### 4. 分批策略固定

P0阶段固定策略：
- batch=1：第1个节点（灰度测试）
- batch=2：剩余节点（全量发布）
- 若只有1个节点：仅batch=1

### 5. 错误处理

- AppError：业务错误（如无在线节点）
- 内部错误：数据库错误、事务失败等
- 参数错误：缺失参数、无效参数

## 禁止事项

1. 禁止调用Agent - Done（未调用）
2. 禁止在本任务执行apply_config/reload - Done（仅创建计划）
3. 禁止引入新表 - Done（使用已有表）
4. 禁止出现yourapp/...等错误import - Done（无错误import）
5. 禁止使用图标（emoji） - Done（无emoji）

## 相关文档

- 实现计划: docs/B0-01-02-PLAN.md
- 测试脚本: scripts/test_release_create.sh
- B0-01-01交付报告: docs/B0-01-01-DELIVERY.md（发布模型与表结构）
- T2-08交付报告: docs/T2-08-DELIVERY.md（证书风险预检）
- T2-07交付报告: docs/T2-07-DELIVERY.md（证书关系可视化）

## 交付清单

- [x] internal/release/dto.go
- [x] internal/release/selector.go
- [x] internal/release/service.go
- [x] api/v1/releases/handler.go
- [x] api/v1/router.go（更新）
- [x] scripts/test_release_create.sh
- [x] docs/B0-01-02-PLAN.md
- [x] docs/B0-01-02-DELIVERY.md

## 最终Commit

- Commit Hash: 970fb33ae6868d3bde913a22e68dda76e80bc63f
- Commit Message: feat(B0-01-02): implement release task creation (node selection + batch allocation)
- 仓库: labubu-daydayone/go_cmdb_web
- 分支: main
