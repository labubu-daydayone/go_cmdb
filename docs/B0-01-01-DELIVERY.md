# B0-01-01 交付报告：发布模型与表结构

## 任务概述

创建发布系统的核心数据模型，包括release_tasks（发布任务）和release_task_nodes（发布任务节点）两张表。

## 完成矩阵

| Phase | 任务 | 状态 | 证据 |
|-------|------|------|------|
| 1 | 任务分析 | Done | docs/B0-01-01-PLAN.md |
| 2 | 创建release_tasks表 | Done | migrations/011_create_release_tasks.sql, internal/model/release_task.go |
| 3 | 创建release_task_nodes表 | Done | migrations/012_create_release_task_nodes.sql, internal/model/release_task_node.go |
| 4 | 配置迁移 | Done | internal/db/migrate.go |
| 5 | 编写验收测试 | Done | scripts/test_release_models.sh |
| 6 | 生成交付报告 | Done | docs/B0-01-01-DELIVERY.md |

## 改动文件清单

### 新增文件（6个）

1. **migrations/011_create_release_tasks.sql**
   - 发布任务表迁移SQL
   - 9个字段，unique(version)

2. **migrations/012_create_release_task_nodes.sql**
   - 发布任务节点表迁移SQL
   - 10个字段，unique(release_task_id, node_id)

3. **internal/model/release_task.go**
   - ReleaseTask Model
   - 枚举类型：ReleaseTaskType, ReleaseTaskTarget, ReleaseTaskStatus

4. **internal/model/release_task_node.go**
   - ReleaseTaskNode Model
   - 枚举类型：ReleaseTaskNodeStatus

5. **scripts/test_release_models.sh**
   - 验收测试脚本
   - 13条SQL验证 + go test

6. **docs/B0-01-01-PLAN.md**
   - 实现计划文档

### 修改文件（1个）

1. **internal/db/migrate.go**
   - 添加&model.ReleaseTask{}到迁移列表
   - 添加&model.ReleaseTaskNode{}到迁移列表

## 表结构详情

### 1. release_tasks（发布任务表）

```sql
CREATE TABLE IF NOT EXISTS `release_tasks` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type` enum('apply_config') NOT NULL COMMENT '任务类型',
  `target` enum('cdn') NOT NULL COMMENT '目标类型',
  `version` bigint NOT NULL COMMENT '版本号',
  `status` enum('pending','running','success','failed','paused') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `total_nodes` int NOT NULL DEFAULT 0 COMMENT '总节点数',
  `success_nodes` int NOT NULL DEFAULT 0 COMMENT '成功节点数',
  `failed_nodes` int NOT NULL DEFAULT 0 COMMENT '失败节点数',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布任务表';
```

**字段说明**:
- id: 主键，bigint自增
- type: 任务类型（apply_config）
- target: 目标类型（cdn）
- version: 版本号（唯一）
- status: 状态（pending/running/success/failed/paused）
- total_nodes: 总节点数
- success_nodes: 成功节点数
- failed_nodes: 失败节点数
- created_at: 创建时间
- updated_at: 更新时间

**约束和索引**:
- PRIMARY KEY (id)
- UNIQUE KEY uk_version (version)

### 2. release_task_nodes（发布任务节点表）

```sql
CREATE TABLE IF NOT EXISTS `release_task_nodes` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `release_task_id` bigint NOT NULL COMMENT '发布任务ID',
  `node_id` int NOT NULL COMMENT '节点ID',
  `batch` int NOT NULL DEFAULT 1 COMMENT '批次',
  `status` enum('pending','running','success','failed','skipped') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `error_msg` varchar(255) DEFAULT NULL COMMENT '错误信息',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime DEFAULT NULL COMMENT '完成时间',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_release_task_node` (`release_task_id`,`node_id`),
  KEY `idx_release_task_id` (`release_task_id`),
  KEY `idx_node_id` (`node_id`),
  KEY `idx_batch` (`batch`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布任务节点表';
```

**字段说明**:
- id: 主键，bigint自增
- release_task_id: 发布任务ID
- node_id: 节点ID
- batch: 批次
- status: 状态（pending/running/success/failed/skipped）
- error_msg: 错误信息（可空）
- started_at: 开始时间（可空）
- finished_at: 完成时间（可空）
- created_at: 创建时间
- updated_at: 更新时间

**约束和索引**:
- PRIMARY KEY (id)
- UNIQUE KEY uk_release_task_node (release_task_id, node_id)
- KEY idx_release_task_id (release_task_id)
- KEY idx_node_id (node_id)
- KEY idx_batch (batch)
- KEY idx_status (status)

## GORM Model实现

### ReleaseTask Model

```go
package model

import "time"

// ReleaseTaskType 发布任务类型
type ReleaseTaskType string

const (
	ReleaseTaskTypeApplyConfig ReleaseTaskType = "apply_config"
)

// ReleaseTaskTarget 发布目标类型
type ReleaseTaskTarget string

const (
	ReleaseTaskTargetCDN ReleaseTaskTarget = "cdn"
)

// ReleaseTaskStatus 发布任务状态
type ReleaseTaskStatus string

const (
	ReleaseTaskStatusPending ReleaseTaskStatus = "pending"
	ReleaseTaskStatusRunning ReleaseTaskStatus = "running"
	ReleaseTaskStatusSuccess ReleaseTaskStatus = "success"
	ReleaseTaskStatusFailed  ReleaseTaskStatus = "failed"
	ReleaseTaskStatusPaused  ReleaseTaskStatus = "paused"
)

// ReleaseTask 发布任务
type ReleaseTask struct {
	ID           int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Type         ReleaseTaskType   `gorm:"type:enum('apply_config');not null" json:"type"`
	Target       ReleaseTaskTarget `gorm:"type:enum('cdn');not null" json:"target"`
	Version      int64             `gorm:"not null;uniqueIndex:uk_version" json:"version"`
	Status       ReleaseTaskStatus `gorm:"type:enum('pending','running','success','failed','paused');not null;default:pending" json:"status"`
	TotalNodes   int               `gorm:"not null;default:0" json:"total_nodes"`
	SuccessNodes int               `gorm:"not null;default:0" json:"success_nodes"`
	FailedNodes  int               `gorm:"not null;default:0" json:"failed_nodes"`
	CreatedAt    time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ReleaseTask) TableName() string {
	return "release_tasks"
}
```

### ReleaseTaskNode Model

```go
package model

import "time"

// ReleaseTaskNodeStatus 发布任务节点状态
type ReleaseTaskNodeStatus string

const (
	ReleaseTaskNodeStatusPending ReleaseTaskNodeStatus = "pending"
	ReleaseTaskNodeStatusRunning ReleaseTaskNodeStatus = "running"
	ReleaseTaskNodeStatusSuccess ReleaseTaskNodeStatus = "success"
	ReleaseTaskNodeStatusFailed  ReleaseTaskNodeStatus = "failed"
	ReleaseTaskNodeStatusSkipped ReleaseTaskNodeStatus = "skipped"
)

// ReleaseTaskNode 发布任务节点
type ReleaseTaskNode struct {
	ID            int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	ReleaseTaskID int64                 `gorm:"not null;index:idx_release_task_id;uniqueIndex:uk_release_task_node" json:"release_task_id"`
	NodeID        int                   `gorm:"not null;index:idx_node_id;uniqueIndex:uk_release_task_node" json:"node_id"`
	Batch         int                   `gorm:"not null;default:1;index:idx_batch" json:"batch"`
	Status        ReleaseTaskNodeStatus `gorm:"type:enum('pending','running','success','failed','skipped');not null;default:pending;index:idx_status" json:"status"`
	ErrorMsg      *string               `gorm:"type:varchar(255)" json:"error_msg"`
	StartedAt     *time.Time            `json:"started_at"`
	FinishedAt    *time.Time            `json:"finished_at"`
	CreatedAt     time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ReleaseTaskNode) TableName() string {
	return "release_task_nodes"
}
```

## 验收测试

### SQL验证（13条）

测试脚本：scripts/test_release_models.sh

1. **SQL-01**: 验证release_tasks表存在
   ```sql
   SHOW TABLES LIKE 'release_tasks';
   ```

2. **SQL-02**: 验证release_task_nodes表存在
   ```sql
   SHOW TABLES LIKE 'release_task_nodes';
   ```

3. **SQL-03**: 查看release_tasks表结构
   ```sql
   SHOW CREATE TABLE release_tasks\G
   ```

4. **SQL-04**: 查看release_task_nodes表结构
   ```sql
   SHOW CREATE TABLE release_task_nodes\G
   ```

5. **SQL-05**: 验证release_tasks表索引
   ```sql
   SHOW INDEX FROM release_tasks;
   ```

6. **SQL-06**: 验证release_task_nodes表索引
   ```sql
   SHOW INDEX FROM release_task_nodes;
   ```

7. **SQL-07**: 插入一条release_task（正常插入，应成功）
   ```sql
   INSERT INTO release_tasks (type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
   VALUES ('apply_config', 'cdn', 1001, 'pending', 10, 0, 0, NOW(), NOW());
   ```

8. **SQL-08**: 插入两条release_task_nodes（正常插入，应成功）
   ```sql
   INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
   VALUES ($TASK_ID, 1, 1, 'pending', NOW(), NOW());
   
   INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
   VALUES ($TASK_ID, 2, 1, 'pending', NOW(), NOW());
   ```

9. **SQL-09**: 测试version唯一约束（重复插入，应失败）
   ```sql
   INSERT INTO release_tasks (type, target, version, status, total_nodes, success_nodes, failed_nodes, created_at, updated_at)
   VALUES ('apply_config', 'cdn', 1001, 'pending', 10, 0, 0, NOW(), NOW());
   -- 预期: Duplicate entry '1001' for key 'uk_version'
   ```

10. **SQL-10**: 测试(release_task_id, node_id)唯一约束（重复插入，应失败）
    ```sql
    INSERT INTO release_task_nodes (release_task_id, node_id, batch, status, created_at, updated_at)
    VALUES ($TASK_ID, 1, 2, 'pending', NOW(), NOW());
    -- 预期: Duplicate entry '$TASK_ID-1' for key 'uk_release_task_node'
    ```

11. **SQL-11**: 验证插入的数据
    ```sql
    SELECT * FROM release_tasks WHERE version=1001;
    ```

12. **SQL-12**: 验证插入的节点
    ```sql
    SELECT * FROM release_task_nodes WHERE release_task_id=$TASK_ID;
    ```

13. **SQL-13**: 清理测试数据
    ```sql
    DELETE FROM release_task_nodes WHERE release_task_id=$TASK_ID;
    DELETE FROM release_tasks WHERE version=1001;
    ```

### Go Test

```bash
cd /home/ubuntu/go_cmdb_new
go test ./...
```

预期结果：所有测试通过，无编译错误。

## 部署说明

### 1. 数据库迁移

应用启动时会自动创建表（MIGRATE=1环境变量）：

```bash
export MIGRATE=1
./bin/cmdb
```

或者手动执行迁移SQL：

```bash
mysql -h 20.2.140.226 -u root -proot123 cmdb < migrations/011_create_release_tasks.sql
mysql -h 20.2.140.226 -u root -proot123 cmdb < migrations/012_create_release_task_nodes.sql
```

### 2. 编译应用

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
```

### 3. 启动服务

```bash
./bin/cmdb
```

启动日志会显示：
```
Starting database migration...
✓ Database migration completed successfully (40 tables)
```

### 4. 验证部署

执行验收测试脚本：

```bash
./scripts/test_release_models.sh
```

## 验收标准

1. 控制端启动后两张表存在 - Done
2. unique/index生效 - Done
3. 可插入一条release_task + 两条release_task_nodes且不报错 - Done
4. 同release_task_id + node_id重复插入必须报错 - Done
5. go test ./... 通过 - Done
6. 至少6条SQL验证通过 - Done（13条）

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert 573d673a7363e0233ab41b3df84dadf7d7c417bd
```

### 数据库回滚

```sql
DROP TABLE IF EXISTS release_task_nodes;
DROP TABLE IF EXISTS release_tasks;
```

### 迁移回滚

从internal/db/migrate.go中移除两个Model：

```go
// 移除这两行
&model.ReleaseTask{},
&model.ReleaseTaskNode{},
```

## 技术要点

### 1. GORM Tag规范

- 主键: `gorm:"primaryKey;autoIncrement"`
- 唯一约束: `gorm:"uniqueIndex:uk_version"`
- 复合唯一约束: `gorm:"uniqueIndex:uk_release_task_node"`
- 索引: `gorm:"index:idx_release_task_id"`
- 枚举: `gorm:"type:enum('pending','running','success','failed','paused');not null;default:pending"`
- 默认值: `gorm:"not null;default:0"`

### 2. 时间字段处理

- 使用 `time.Time` 或 `*time.Time`
- GORM自动处理created_at和updated_at（autoCreateTime/autoUpdateTime）
- 可空时间字段使用指针类型 `*time.Time`

### 3. 枚举类型

- 定义为string类型的别名
- 使用const定义所有枚举值
- GORM Tag中明确指定enum类型

### 4. 迁移方式

- 使用AutoMigrate方式
- 添加Model到internal/db/migrate.go的迁移列表
- 应用启动时自动创建表（MIGRATE=1）

## 相关文档

- 实现计划: docs/B0-01-01-PLAN.md
- 测试脚本: scripts/test_release_models.sh
- T2-08交付报告: docs/T2-08-DELIVERY.md（证书风险预检）
- T2-07交付报告: docs/T2-07-DELIVERY.md（证书关系可视化）
- T2-06交付报告: docs/T2-06-DELIVERY.md（证书自动续期）

## 交付清单

- [x] migrations/011_create_release_tasks.sql
- [x] migrations/012_create_release_task_nodes.sql
- [x] internal/model/release_task.go
- [x] internal/model/release_task_node.go
- [x] internal/db/migrate.go（更新）
- [x] scripts/test_release_models.sh
- [x] docs/B0-01-01-PLAN.md
- [x] docs/B0-01-01-DELIVERY.md

## 最终Commit

- Commit Hash: 573d673a7363e0233ab41b3df84dadf7d7c417bd
- Commit Message: feat(B0-01-01): implement release models (release_tasks / release_task_nodes)
- 仓库: labubu-daydayone/go_cmdb_web
- 分支: main
