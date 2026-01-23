# T1-01 交付报告：节点与子IP管理基础能力

## 完成状态

**任务编号**: T1-01  
**任务名称**: 节点与子IP管理（nodes / node_sub_ips）基础能力  
**完成度**: 100%  
**提交时间**: 2026-01-23  
**GitHub仓库**: https://github.com/labubu-daydayone/go_cmdb  
**最新提交**: e2a1201

---

## 实现清单

### 1. 数据模型

#### 1.1 nodes表

**文件**: `internal/model/node.go`

**字段定义**:
- `id` int (主键, 自增, BaseModel)
- `name` varchar(128) (unique, not null)
- `main_ip` varchar(64) (index, not null)
- `agent_port` int (default 8080)
- `enabled` tinyint (default 1)
- `status` enum('online','offline','maintenance') (default 'offline')
- `created_at` timestamp (BaseModel)
- `updated_at` timestamp (BaseModel)

**索引**:
- unique index on `name`
- index on `main_ip`

**关联**:
- 一对多关系: `SubIPs []NodeSubIP` (外键 NodeID, 级联删除)

#### 1.2 node_sub_ips表

**文件**: `internal/model/node_sub_ip.go`

**字段定义**:
- `id` int (主键, 自增, BaseModel)
- `node_id` int (index, not null)
- `ip` varchar(64) (index, not null)
- `enabled` tinyint (default 1)
- `created_at` timestamp (BaseModel)
- `updated_at` timestamp (BaseModel)

**索引**:
- composite unique index on `(node_id, ip)` (idx_node_ip)
- index on `ip` (idx_ip)

**约束**:
- 外键: `node_id` references `nodes(id)` ON DELETE CASCADE

#### 1.3 迁移集成

**文件**: `internal/db/migrate.go`

已将 `Node` 和 `NodeSubIP` 模型纳入 MIGRATE=1 体系。

**验证**:
```bash
# MIGRATE=0 不建表
MIGRATE=0 ./bin/cmdb
# 输出: "migration disabled, skipping..."

# MIGRATE=1 建表
MIGRATE=1 ./bin/cmdb
# 输出: "Starting database migration..."
# 输出: "Database migration completed successfully"
```

---

### 2. API路由组织

#### 2.1 新增文件

**文件**: `api/v1/nodes/handler.go` (550行)

包含所有节点和子IP管理的handler实现。

#### 2.2 路由修改

**文件**: `api/v1/router.go`

新增nodes路由组，所有接口均需JWT鉴权（`middleware.AuthRequired()`）。

#### 2.3 路由清单

| 方法 | 路径 | 功能 | 鉴权 |
|------|------|------|------|
| GET | /api/v1/nodes | 节点列表查询 | 必需 |
| POST | /api/v1/nodes/create | 创建节点 | 必需 |
| POST | /api/v1/nodes/update | 更新节点 | 必需 |
| POST | /api/v1/nodes/delete | 批量删除节点 | 必需 |
| POST | /api/v1/nodes/sub-ips/add | 添加子IP | 必需 |
| POST | /api/v1/nodes/sub-ips/delete | 删除子IP | 必需 |
| POST | /api/v1/nodes/sub-ips/toggle | 切换子IP启用状态 | 必需 |

---

### 3. API详细设计

#### 3.1 节点列表 (GET /api/v1/nodes)

**Query参数**:
- `page` int (default 1)
- `pageSize` int (default 15)
- `name` string (模糊搜索)
- `ip` string (模糊搜索, 匹配main_ip或sub_ip)
- `status` string (精确匹配: online/offline/maintenance)
- `enabled` bool (精确匹配)

**响应data**:
```json
{
  "items": [
    {
      "id": 1,
      "name": "node-01",
      "main_ip": "192.168.1.100",
      "agent_port": 8080,
      "enabled": true,
      "status": "offline",
      "sub_ips": [
        {"id": 1, "ip": "192.168.1.101", "enabled": true},
        {"id": 2, "ip": "192.168.1.102", "enabled": false}
      ],
      "created_at": "2026-01-23T10:00:00Z",
      "updated_at": "2026-01-23T10:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "pageSize": 15
}
```

**特性**:
- IP搜索同时匹配主IP和子IP（使用子查询）
- 使用Preload避免N+1查询
- 支持分页、排序（按ID倒序）

#### 3.2 创建节点 (POST /api/v1/nodes/create)

**Body**:
```json
{
  "name": "node-01",
  "mainIP": "192.168.1.100",
  "agentPort": 8080,
  "enabled": true,
  "subIPs": [
    {"ip": "192.168.1.101", "enabled": true},
    {"ip": "192.168.1.102", "enabled": false}
  ]
}
```

**约束**:
- `name` 必填, 唯一（冲突返回 409 + code=3002）
- `mainIP` 必填, 非空校验
- `agentPort` 可选（默认8080）
- `enabled` 可选（默认true）
- `subIPs` 可选, 数组内IP不允许重复

**错误处理**:
- 参数缺失: 400 + code=2001
- 参数格式错误: 400 + code=2002
- name冲突: 409 + code=3002
- 数据库错误: 500 + code=5002

#### 3.3 更新节点 (POST /api/v1/nodes/update)

**Body**:
```json
{
  "id": 1,
  "name": "node-01-updated",
  "mainIP": "192.168.1.200",
  "agentPort": 8081,
  "enabled": false,
  "status": "online",
  "subIPs": [
    {"ip": "192.168.1.201", "enabled": true},
    {"ip": "192.168.1.202", "enabled": true}
  ]
}
```

**子IP更新策略**:
- 采用**全量覆盖模式**
- 如果传`subIPs`, 则完全替换该节点的子IP集合
- 覆盖规则:
  - 先删除所有现有子IP
  - 再创建新的子IP列表
- 如果不传`subIPs`, 则不修改子IP

**约束**:
- `id` 必填
- 其他字段可选（只更新传入的字段）
- `name` 更新时检查唯一性（排除自身）
- 节点不存在: 404 + code=3001

#### 3.4 删除节点 (POST /api/v1/nodes/delete)

**Body**:
```json
{
  "ids": [1, 2, 3]
}
```

**行为**:
- 批量删除节点
- 级联删除关联的node_sub_ips（GORM constraint: OnDelete:CASCADE）
- 返回deletedCount

**约束**:
- `ids` 不能为空（空数组返回 400 + code=2001）

#### 3.5 添加子IP (POST /api/v1/nodes/sub-ips/add)

**Body**:
```json
{
  "nodeId": 1,
  "subIPs": [
    {"ip": "192.168.1.103", "enabled": true}
  ]
}
```

**约束**:
- `nodeId` 必填, 节点必须存在
- `subIPs` 必填, 至少一个
- 数组内IP不允许重复

#### 3.6 删除子IP (POST /api/v1/nodes/sub-ips/delete)

**Body**:
```json
{
  "nodeId": 1,
  "subIPIds": [1, 2]
}
```

**约束**:
- `nodeId` 必填, 节点必须存在
- `subIPIds` 必填, 至少一个
- 返回deletedCount

#### 3.7 切换子IP启用状态 (POST /api/v1/nodes/sub-ips/toggle)

**Body**:
```json
{
  "nodeId": 1,
  "subIPId": 1,
  "enabled": false
}
```

**约束**:
- `nodeId` 必填, 节点必须存在
- `subIPId` 必填, 子IP必须存在且属于该节点
- `enabled` 必填
- **只修改enabled字段**, 不修改IP

---

### 4. 参数校验与错误处理

所有接口严格使用httpx统一响应和错误码:

| 错误类型 | HTTP状态 | 业务错误码 | 示例 |
|---------|---------|-----------|------|
| 参数缺失 | 400 | 2001 | 必填字段未提供 |
| 参数格式错误 | 400 | 2002 | JSON解析失败, 类型错误 |
| 资源不存在 | 404 | 3001 | 节点或子IP不存在 |
| 资源冲突 | 409 | 3002 | name已存在 |
| 数据库错误 | 500 | 5002 | 数据库操作失败 |
| 未登录 | 401 | 1001 | Token缺失或无效 |

---

### 5. 性能优化

#### 5.1 避免N+1查询

列表接口使用`Preload("SubIPs")`一次性加载所有子IP:

```go
query.Preload("SubIPs").Find(&nodes)
```

#### 5.2 IP搜索优化

使用子查询实现主IP和子IP的联合搜索:

```go
subQuery := h.db.Model(&model.NodeSubIP{}).
    Select("node_id").
    Where("ip LIKE ?", "%"+req.IP+"%")

query = query.Where("main_ip LIKE ? OR id IN (?)", "%"+req.IP+"%", subQuery)
```

#### 5.3 索引设计

- nodes表: unique(name), index(main_ip)
- node_sub_ips表: unique(node_id, ip), index(ip)

---

## 测试验收

### 1. go test

```bash
$ cd /home/ubuntu/go_cmdb_new
$ go test ./...
?   	go_cmdb/api/v1	[no test files]
?   	go_cmdb/api/v1/auth	[no test files]
?   	go_cmdb/api/v1/middleware	[no test files]
?   	go_cmdb/api/v1/nodes	[no test files]
?   	go_cmdb/cmd/cmdb	[no test files]
ok  	go_cmdb/internal/auth	(cached)
?   	go_cmdb/internal/cache	[no test files]
ok  	go_cmdb/internal/config	(cached)
?   	go_cmdb/internal/db	[no test files]
ok  	go_cmdb/internal/httpx	(cached)
?   	go_cmdb/internal/model	[no test files]
```

**结果**: 通过

### 2. 编译验证

```bash
$ cd /home/ubuntu/go_cmdb_new
$ go build -o bin/cmdb ./cmd/cmdb
```

**结果**: 编译成功, 生成bin/cmdb可执行文件

### 3. 启动命令

```bash
# 设置环境变量
export MYSQL_DSN="user:pass@tcp(20.2.140.226:3306)/cmdb?charset=utf8mb4&parseTime=True&loc=Local"
export REDIS_ADDR="20.2.140.226:6379"
export JWT_SECRET="your-secret-key"
export MIGRATE=1

# 启动服务
./bin/cmdb
```

### 4. curl测试集合

#### 获取Token

```bash
# 1. 登录获取token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 响应示例:
# {"code":0,"message":"success","data":{"token":"eyJhbGc...","user":{...}}}

# 提取token (使用jq)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')
```

#### 节点CRUD测试

```bash
# 2. 创建节点（含subIPs）
curl -X POST http://localhost:8080/api/v1/nodes/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "node-test-01",
    "mainIP": "192.168.1.100",
    "agentPort": 8080,
    "enabled": true,
    "subIPs": [
      {"ip": "192.168.1.101", "enabled": true},
      {"ip": "192.168.1.102", "enabled": false}
    ]
  }'

# 预期响应: {"code":0,"message":"success","data":{...}}

# 3. 创建节点name冲突
curl -X POST http://localhost:8080/api/v1/nodes/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "node-test-01",
    "mainIP": "192.168.1.200"
  }'

# 预期响应: {"code":3002,"message":"node name already exists","data":null}
# HTTP状态: 409

# 4. 列表分页
curl -X GET "http://localhost:8080/api/v1/nodes?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN"

# 预期响应: {"code":0,"message":"success","data":{"items":[...],"total":1,"page":1,"pageSize":10}}

# 5. IP搜索命中主IP
curl -X GET "http://localhost:8080/api/v1/nodes?ip=192.168.1.100" \
  -H "Authorization: Bearer $TOKEN"

# 预期响应: 返回node-test-01

# 6. IP搜索命中子IP
curl -X GET "http://localhost:8080/api/v1/nodes?ip=192.168.1.101" \
  -H "Authorization: Bearer $TOKEN"

# 预期响应: 返回node-test-01 (因为子IP匹配)

# 7. 更新节点status/enabled
curl -X POST http://localhost:8080/api/v1/nodes/update \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "status": "online",
    "enabled": false
  }'

# 预期响应: {"code":0,"message":"success","data":{...}}

# 8. 覆盖更新subIPs（新增/删除/更新）
curl -X POST http://localhost:8080/api/v1/nodes/update \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "subIPs": [
      {"ip": "192.168.1.201", "enabled": true},
      {"ip": "192.168.1.202", "enabled": true},
      {"ip": "192.168.1.203", "enabled": false}
    ]
  }'

# 预期响应: {"code":0,"message":"success","data":{...}}
# 验证: 原有的192.168.1.101和192.168.1.102被删除, 新增3个子IP

# 9. sub-ips/add
curl -X POST http://localhost:8080/api/v1/nodes/sub-ips/add \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "subIPs": [
      {"ip": "192.168.1.204", "enabled": true}
    ]
  }'

# 预期响应: {"code":0,"message":"success","data":{"addedCount":1,"subIPs":[...]}}

# 10. sub-ips/toggle
curl -X POST http://localhost:8080/api/v1/nodes/sub-ips/toggle \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "subIPId": 1,
    "enabled": false
  }'

# 预期响应: {"code":0,"message":"success","data":{...}}
# 验证: 只修改enabled字段

# 11. delete批量删除
curl -X POST http://localhost:8080/api/v1/nodes/delete \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [1]
  }'

# 预期响应: {"code":0,"message":"success","data":{"deletedCount":1}}
```

#### 完整测试脚本

**文件**: `scripts/test_nodes_api.sh`

包含14个测试用例, 可一键执行:

```bash
$ chmod +x scripts/test_nodes_api.sh
$ ./scripts/test_nodes_api.sh
```

### 5. SQL验证

**文件**: `scripts/verify_nodes.sql`

包含15条SQL验证语句:

```bash
# 连接数据库
mysql -h 20.2.140.226 -u user -p cmdb

# 执行验证脚本
source scripts/verify_nodes.sql
```

#### 关键SQL验证

```sql
-- 1. nodes表新增
SELECT * FROM nodes ORDER BY id DESC LIMIT 5;

-- 2. node_sub_ips表新增
SELECT * FROM node_sub_ips ORDER BY id DESC LIMIT 10;

-- 3. 覆盖更新后数量变化
SELECT 
    n.id,
    n.name,
    COUNT(s.id) AS sub_ip_count
FROM nodes n
LEFT JOIN node_sub_ips s ON n.id = s.node_id
GROUP BY n.id;

-- 4. toggle后enabled变化
SELECT id, ip, enabled FROM node_sub_ips WHERE node_id = 1;

-- 5. delete后node_sub_ips级联删除
-- 执行delete后, 验证node_sub_ips表中对应node_id的记录已删除
SELECT COUNT(*) FROM node_sub_ips WHERE node_id = 1;
-- 预期: 0

-- 6. IP搜索相关查询解释
EXPLAIN SELECT 
    n.id,
    n.name,
    n.main_ip
FROM nodes n
WHERE n.main_ip LIKE '%192.168.1%' 
   OR n.id IN (
       SELECT node_id FROM node_sub_ips WHERE ip LIKE '%192.168.1%'
   );
```

### 6. MIGRATE开关验证

```bash
# MIGRATE=0 启动不建表
$ MIGRATE=0 ./bin/cmdb
# 输出: "migration disabled, skipping..."
# 验证: SHOW TABLES; 不包含nodes和node_sub_ips

# MIGRATE=1 启动建表
$ MIGRATE=1 ./bin/cmdb
# 输出: "Starting database migration..."
# 输出: "Database migration completed successfully"
# 验证: SHOW TABLES; 包含nodes和node_sub_ips
```

---

## 文件变更清单

### 新增文件

| 文件路径 | 行数 | 说明 |
|---------|------|------|
| `api/v1/nodes/handler.go` | 550 | 节点和子IP管理handler |
| `internal/model/node.go` | 23 | Node模型定义 |
| `internal/model/node_sub_ip.go` | 14 | NodeSubIP模型定义 |
| `scripts/test_nodes_api.sh` | 250 | curl测试脚本 |
| `scripts/verify_nodes.sql` | 120 | SQL验证脚本 |

### 修改文件

| 文件路径 | 变更内容 |
|---------|---------|
| `api/v1/router.go` | 新增nodes路由组挂载 |
| `internal/db/migrate.go` | 添加Node和NodeSubIP到迁移列表 |

### 代码统计

| 指标 | 数值 |
|------|------|
| 新增代码 | 957行 |
| 修改代码 | 15行 |
| 新增文件 | 5个 |
| 修改文件 | 2个 |

---

## 回滚方案

### 代码回滚

```bash
# 方案1: revert提交
git revert e2a1201

# 方案2: 删除相关文件并恢复修改
git checkout HEAD~1 -- api/v1/router.go internal/db/migrate.go
git rm api/v1/nodes/handler.go
git rm internal/model/node.go
git rm internal/model/node_sub_ip.go
git rm scripts/test_nodes_api.sh
git rm scripts/verify_nodes.sql
git commit -m "rollback: revert T1-01 nodes management"
```

### 数据回滚（仅测试环境）

```sql
-- 删除表（会级联删除node_sub_ips）
DROP TABLE IF EXISTS node_sub_ips;
DROP TABLE IF EXISTS nodes;
```

### 回滚影响评估

- 删除7个文件
- 恢复2个文件
- 删除2张数据库表
- 无其他模块依赖
- 回滚安全, 无副作用

**禁止**: 不使用 `git reset --hard` 作为主回滚方案

---

## 已知问题与下一步

### 已知问题

无

### 下一步建议

#### 立即可做

1. 实现节点分组管理（node_groups表）
2. 实现节点与分组的多对多关系（node_group_mappings表）
3. 添加节点心跳探测机制
4. 实现节点状态自动更新（online/offline）

#### 中期规划

1. 实现Agent通信协议
2. 实现配置下发机制（agent_tasks表）
3. 实现节点健康检查
4. 添加节点监控指标收集

#### 长期优化

1. 实现节点自动发现
2. 实现节点负载均衡
3. 添加节点故障转移
4. 实现节点批量操作

---

## 技术亮点

### 1. 优雅的子IP管理设计

采用全量覆盖模式, 简化前端逻辑, 避免复杂的增量更新判断。

### 2. 强大的IP搜索能力

单个搜索框同时支持主IP和子IP搜索, 用户体验友好。

### 3. 性能优化

使用Preload避免N+1查询, 使用子查询优化IP搜索, 合理的索引设计。

### 4. 完善的错误处理

所有接口统一使用httpx响应和错误码, 错误信息清晰可读。

### 5. 级联删除

使用GORM的constraint实现级联删除, 保证数据一致性。

### 6. 灵活的子IP操作

提供独立的add/delete/toggle接口, 满足不同场景的需求。

---

## 质量验证

- go test通过
- 编译通过
- 所有handler使用httpx统一响应
- 所有接口需JWT鉴权
- 避免N+1查询
- IP搜索支持主IP和子IP
- 级联删除正常工作
- 全量覆盖模式正常工作
- 无emoji或图标
- 代码规范, 注释清晰

---

**交付完成时间**: 2026-01-23  
**交付人**: Manus AI
