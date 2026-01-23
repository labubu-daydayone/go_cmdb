# T2-09 交付报告：Websites 列表实时同步（Socket.IO + JWT 认证）

## 任务信息

任务编号：T2-09
任务名称：Websites 列表实时同步（Socket.IO，JWT 统一认证）
开发位置：控制端 go_cmdb（不写前端、不改 Agent）
优先级：P0（可上线最小闭环）
交付日期：2026-01-24

## 完成概览

最终Commit: fc7219a
仓库: labubu-daydayone/go_cmdb_web
状态: 已推送到GitHub main分支

## 完成矩阵（7个Phase全部完成）

### Phase 1: 任务分析 - Done
文件: docs/T2-09-PLAN.md
证据: 完整实现计划

### Phase 2: 创建ws_events表和数据模型 - Done
文件: models/ws_event.go, migrations/008_create_ws_events.sql
证据: 表已创建，包含id/topic/event_type/payload/created_at + 索引

### Phase 3: 实现Socket.IO服务端和JWT握手鉴权 - Done
文件: internal/ws/server.go, internal/ws/auth.go, cmd/cmdb/main.go, api/v1/router.go
证据: Socket.IO服务端已初始化，JWT认证已集成

### Phase 4: 实现websites事件推送和增量补发 - Done
文件: internal/ws/publisher.go, internal/ws/handler.go
证据: PublishWebsiteEvent、GetIncrementalEvents、sendIncrementalUpdates、sendFullWebsitesList

### Phase 5: 集成CRUD联动和事件广播 - Done
文件: api/v1/websites/handler.go
证据: Create/Update/Delete handler中调用ws.PublishWebsiteEvent

### Phase 6: 编写验收测试 - Done
文件: scripts/test_websocket.sh
证据: 18条SQL验证 + 13条curl/Node.js验证

### Phase 7: 生成交付报告 - Done
文件: docs/T2-09-DELIVERY.md
证据: 本报告

## 改动文件清单

### 新增文件（8个）

1. models/ws_event.go - WebSocket事件数据模型
2. migrations/008_create_ws_events.sql - ws_events表迁移脚本
3. internal/ws/server.go - Socket.IO服务端初始化
4. internal/ws/auth.go - JWT握手鉴权
5. internal/ws/handler.go - 事件处理器（request:websites）
6. internal/ws/publisher.go - 事件发布和增量补发
7. scripts/test_websocket.sh - 验收测试脚本
8. docs/T2-09-PLAN.md - 实现计划

### 修改文件（3个）

1. cmd/cmdb/main.go - 添加Socket.IO初始化
2. api/v1/router.go - 挂载Socket.IO到/socket.io/路径
3. api/v1/websites/handler.go - Create/Update/Delete中添加事件广播

## 功能实现

### 1. Socket.IO服务端

文件位置：internal/ws/server.go

功能：
- 使用默认路径：/socket.io/
- 监听地址复用现有HTTP server（Gin同端口）
- 支持polling和websocket两种传输方式
- 连接确认事件：connected { "ok": true }
- 事件处理器：request:websites

### 2. JWT握手鉴权

文件位置：internal/ws/auth.go

功能：
- 客户端通过query参数token或Authorization header传JWT
- 服务端在connection阶段验证JWT
- 验证失败：拒绝连接（401 Unauthorized）
- 复用现有internal/auth/jwt.go的校验函数

### 3. ws_events表

文件位置：migrations/008_create_ws_events.sql

字段：
- id bigint pk auto_increment（作为eventId）
- topic varchar(64) not null（固定填 "websites"）
- event_type enum('add','update','delete') not null
- payload json not null（推送内容，前端可直接用）
- created_at datetime

索引：
- index(topic, id)

### 4. 事件推送和增量补发

文件位置：internal/ws/publisher.go, internal/ws/handler.go

功能：
- PublishWebsiteEvent：写入ws_events表并广播
- GetIncrementalEvents：查询增量事件（最多500条）
- GetLatestEventId：获取最新事件ID
- sendIncrementalUpdates：发送增量更新
- sendFullWebsitesList：发送全量列表

逻辑：
- lastEventId > 0时查询增量事件
- 增量事件 <= 500条：逐条emit websites:update
- 增量事件 > 500条：回落emit websites:initial（全量）
- 无lastEventId：emit websites:initial（全量）

### 5. CRUD联动

文件位置：api/v1/websites/handler.go

功能：
- POST /api/v1/websites/create → 事务成功后调用ws.PublishWebsiteEvent("add", ...)
- POST /api/v1/websites/update → 事务成功后调用ws.PublishWebsiteEvent("update", ...)
- POST /api/v1/websites/delete → 事务成功后调用ws.PublishWebsiteEvent("delete", ...)（支持批量）

要求：
- 必须在DB事务提交成功后再写ws_events
- 广播失败不影响主流程（仅记录日志）

## Socket.IO事件定义

### (1) 连接确认

事件：connected
数据：{ "ok": true }

### (2) 初始全量

客户端发送：request:websites，payload可为空或 { lastEventId?: number }

服务端响应：websites:initial
数据：
```json
{
  "items": [ ...website_list_items... ],
  "total": 123,
  "version": 0,
  "lastEventId": 1024
}
```

items字段：
- id/line_group_id/cache_rule_id
- origin_mode/origin_group_id/origin_set_id
- redirect_url/redirect_status_code
- status/https_enabled
- created_at/updated_at

### (3) 增量更新广播

事件：websites:update
数据：
```json
{
  "eventId": 1025,
  "type": "add|update|delete",
  "data": { "id": 1001 }
}
```

## 验收测试

### SQL验证（18条）

1. 清理测试数据 ✓
2. 验证ws_events表结构 ✓
3. 验证idx_topic_id索引 ✓
4-6. 插入测试事件（add/update/delete）✓
7. 验证事件写入（3条）✓
8. 验证topic过滤 ✓
9-11. 验证event_type过滤（add/update/delete各1条）✓
12-13. 验证按lastEventId拉取（增量查询）✓
14. 验证payload格式（JSON有效）✓
15. 验证created_at字段 ✓
16. 验证最新事件ID查询 ✓
17. 验证事件排序（ASC）✓
18. 验证LIMIT查询（模拟增量补发限制）✓

### curl/Node.js验证（13条）

1. 无token连接被拒绝（401）
2. 正常token连接成功收到connected
3. request:websites收到websites:initial
4. create website后收到websites:update add
5. update website后收到websites:update update
6. delete website后收到websites:update delete
7. lastEventId=旧值 → 收到增量事件
8. lastEventId太旧/事件过多 → 回落收到websites:initial
9. JWT token过期 → 连接被拒绝
10. JWT token无效 → 连接被拒绝
11. 多个客户端同时连接 → 都能收到广播
12. 断线重连 → 使用lastEventId补发
13. 无lastEventId → 收到全量列表

## 编译验证

```bash
cd /home/ubuntu/go_cmdb_new
go build -o bin/cmdb cmd/cmdb/main.go
```

结果：编译成功，二进制文件大小19MB

## 单元测试

```bash
go test ./...
```

结果：所有测试通过（auth包、cert包等）

## 验收标准

- [x] 前端只要连上Socket.IO就能拿到全量并接收增量
- [x] JWT认证统一且可靠
- [x] CRUD变更能实时广播
- [x] lastEventId机制可用（增量或全量兜底）
- [x] go test ./... 通过
- [x] 至少15条SQL验证（实际18条）
- [x] 至少12条验证步骤（实际13条）

## 使用示例

### 前端连接（Node.js）

```javascript
// 安装依赖：npm install socket.io-client
const io = require('socket.io-client');

// 获取JWT token（从登录接口）
const token = 'YOUR_JWT_TOKEN_HERE';

// 连接Socket.IO服务器
const socket = io('http://localhost:8080', {
  auth: { token }
});

// 连接成功
socket.on('connect', () => {
  console.log('Connected:', socket.id);
});

// 收到connected确认
socket.on('connected', (data) => {
  console.log('Connected confirmation:', data);
  
  // 请求websites列表
  socket.emit('request:websites', { lastEventId: 0 });
});

// 收到初始全量
socket.on('websites:initial', (data) => {
  console.log('Websites initial:', data);
});

// 收到增量更新
socket.on('websites:update', (data) => {
  console.log('Websites update:', data);
});

// 连接错误
socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message);
});

// 断开连接
socket.on('disconnect', (reason) => {
  console.log('Disconnected:', reason);
});
```

### 前端连接（浏览器）

```html
<!DOCTYPE html>
<html>
<head>
  <title>WebSocket Test</title>
  <script src="https://cdn.socket.io/4.5.4/socket.io.min.js"></script>
</head>
<body>
  <h1>WebSocket Test</h1>
  <div id="status">Disconnected</div>
  <div id="messages"></div>

  <script>
    const token = 'YOUR_JWT_TOKEN_HERE';
    
    const socket = io('http://localhost:8080', {
      auth: { token }
    });

    socket.on('connect', () => {
      document.getElementById('status').textContent = 'Connected: ' + socket.id;
    });

    socket.on('connected', (data) => {
      console.log('Connected confirmation:', data);
      socket.emit('request:websites', { lastEventId: 0 });
    });

    socket.on('websites:initial', (data) => {
      console.log('Websites initial:', data);
      document.getElementById('messages').innerHTML += '<p>Received initial: ' + data.total + ' websites</p>';
    });

    socket.on('websites:update', (data) => {
      console.log('Websites update:', data);
      document.getElementById('messages').innerHTML += '<p>Update: ' + data.type + ' (eventId: ' + data.eventId + ')</p>';
    });

    socket.on('connect_error', (error) => {
      document.getElementById('status').textContent = 'Error: ' + error.message;
    });

    socket.on('disconnect', (reason) => {
      document.getElementById('status').textContent = 'Disconnected: ' + reason;
    });
  </script>
</body>
</html>
```

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
./scripts/test_websocket.sh
```

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert fc7219a
```

### 关闭Socket.IO挂载

注释以下代码（api/v1/router.go）：
```go
// if ws.Server != nil {
//     r.Any("/socket.io/*any", gin.WrapH(ws.WrapWithAuth(ws.Server)))
// }
```

### 删除ws_events表

```sql
DROP TABLE ws_events;
```

### 删除新增文件

```bash
rm -rf internal/ws
rm models/ws_event.go
rm migrations/008_create_ws_events.sql
rm scripts/test_websocket.sh
rm docs/T2-09-PLAN.md
```

### 恢复修改文件

```bash
git checkout HEAD~1 -- cmd/cmdb/main.go
git checkout HEAD~1 -- api/v1/router.go
git checkout HEAD~1 -- api/v1/websites/handler.go
```

## 已知限制

1. 全量列表查询：限制最多10000条记录（防止内存溢出）
2. 增量补发：限制最多500条事件（超过则回落全量）
3. 事件清理：未实现自动清理策略（建议手工清理或定期cron）
4. 房间隔离：未实现room机制（所有客户端在同一命名空间）
5. 列表项字段：部分字段未完整实现（domains、line_group_name、origin_group_name等）

## 后续改进建议

### 短期（1-2周）

- 实现完整的WebsiteListItem字段（domains、line_group_name等）
- 添加房间隔离机制（room = "websites"）
- 实现事件清理策略（保留7天或最大100万条）

### 中期（1-2个月）

- 实现多topic支持（domains、nodes等）
- 实现连接数限制和流量控制
- 添加WebSocket监控和统计

### 长期（3-6个月）

- 实现分布式WebSocket（多实例支持）
- 实现消息持久化和可靠投递
- 实现WebSocket集群和负载均衡

## 相关文档

- 完整交付报告: docs/T2-09-DELIVERY.md
- 实现计划: docs/T2-09-PLAN.md
- 测试脚本: scripts/test_websocket.sh
- Socket.IO官方文档: https://socket.io/docs/v4/

## 注意事项

1. Socket.IO默认路径：/socket.io/（不要使用自定义路径）
2. JWT token通过query参数token或Authorization header传递
3. 连接失败：检查JWT token是否有效
4. 事件广播失败：不影响主流程（仅记录日志）
5. 增量补发：lastEventId太旧时自动回落全量
6. 全量列表：限制最多10000条记录
7. 事件清理：建议定期清理ws_events表（保留7天或最大100万条）

## 交付清单

- [x] Socket.IO服务端（默认路径/socket.io/）
- [x] JWT握手鉴权（复用现有auth逻辑）
- [x] ws_events表（id/topic/event_type/payload/created_at + 索引）
- [x] 事件推送和增量补发（PublishWebsiteEvent、GetIncrementalEvents）
- [x] CRUD联动（Create/Update/Delete中调用事件发布）
- [x] 验收测试脚本（18条SQL + 13条curl/Node.js）
- [x] 交付报告
- [x] 代码提交并推送到GitHub

所有Phase已完成，系统已交付并推送到GitHub。
