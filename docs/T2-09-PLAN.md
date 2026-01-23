# T2-09 实现计划：Websites 列表实时同步（Socket.IO + JWT 认证）

## 任务信息

- 任务编号：T2-09
- 任务名称：Websites 列表实时同步（Socket.IO，JWT 统一认证）
- 开发位置：控制端 go_cmdb（不写前端、不改 Agent）
- 优先级：P0（可上线最小闭环）

## 目标

让前端"网站列表页"实现实时刷新：
- 连接后拿全量 websites:initial
- 网站 create/update/delete 后广播 websites:update
- 支持 lastEventId 增量补发（补不出来就回全量）

## 边界

本任务做：
- Socket.IO 服务端
- JWT 握手鉴权
- websites 事件推送
- 事件落库/补发

本任务不做：
- 前端页面
- 发布系统
- DNS/证书/agent 调用
- 通知/告警

## Phase 1: 分析任务需求并创建实现计划

### 输入
- T2-09 任务需求文档

### 输出
- docs/T2-09-PLAN.md（本文档）

### 任务分解
1. 阅读任务需求文档
2. 分析技术方案
3. 创建实现计划

## Phase 2: 创建 ws_events 表和数据模型

### 输入
- 现有数据库结构

### 输出
- models/ws_event.go（数据模型）
- 数据库迁移 SQL

### 任务分解
1. 创建 ws_events 表（P0 必须）
   - id bigint pk auto_increment（作为 eventId）
   - topic varchar(64) not null（固定填 "websites"）
   - event_type enum('add','update','delete') not null
   - payload json not null（推送内容，前端可直接用）
   - created_at datetime
   - 索引：index(topic, id)

2. 创建 GORM 模型
   - models/ws_event.go

## Phase 3: 实现 Socket.IO 服务端和 JWT 握手鉴权

### 输入
- 现有 JWT 认证逻辑（internal/auth/jwt.go）
- 现有 HTTP server（Gin）

### 输出
- internal/ws/server.go（Socket.IO server 初始化、注册事件）
- internal/ws/auth.go（握手鉴权）
- api/v1/router.go（把 socket.io handler 挂到 http server）

### 任务分解
1. 安装 Socket.IO Go 库
   - github.com/googollee/go-socket.io

2. 实现 Socket.IO server
   - 使用默认路径：/socket.io/
   - 监听地址复用现有 HTTP server（Gin 同端口）

3. 实现 JWT 握手鉴权
   - 客户端通过 auth.token 传 JWT
   - 服务端在 connection 阶段验证 JWT
   - 验证失败：拒绝连接（断开）
   - 复用现有 internal/auth/jwt.go 的校验函数

4. 实现连接确认事件
   - event：connected
   - data：{ "ok": true }

## Phase 4: 实现 websites 事件推送和增量补发

### 输入
- ws_events 表
- Socket.IO server

### 输出
- internal/ws/handler.go（事件处理）
- internal/ws/publisher.go（事件发布）

### 任务分解
1. 实现初始全量推送
   - 客户端发送：request:websites，payload 可为空或 { lastEventId?: number }
   - 服务端响应：websites:initial
   - data：
     ```json
     {
       "items": [ ...website_list_items... ],
       "total": 123,
       "version": 456789,
       "lastEventId": 1024
     }
     ```
   - items 字段要求：
     - website_id
     - primary_domain（从 website_domains.is_primary=1 取）
     - cname（冗余字段或从 line_group 推导）
     - line_group（或 line_group_id + name）
     - https_enabled
     - status

2. 实现增量更新广播
   - event：websites:update
   - data：
     ```json
     {
       "eventId": 1025,
       "type": "add|update|delete",
       "data": { ... }
     }
     ```

3. 实现增量补发（P0 必须）
   - 当客户端发送：request:websites + { lastEventId: 1000 }
   - 服务端逻辑：
     - 查 ws_events where topic='websites' and id > lastEventId order by id asc limit N（N=500）
     - 若查到的事件数量 <= N：逐条 emit websites:update（按顺序）
     - 若事件过多/超过 N：直接返回一次 websites:initial

## Phase 5: 集成 CRUD 联动和事件广播

### 输入
- 现有 websites CRUD API（api/v1/websites/handler.go）
- internal/ws/publisher.go

### 输出
- 修改 api/v1/websites/handler.go（调用事件发布）

### 任务分解
1. 在以下 HTTP API 成功提交事务后写入事件表并广播：
   - POST /api/v1/websites/create → add
   - POST /api/v1/websites/update → update
   - POST /api/v1/websites/delete → delete

2. 要求：
   - 必须在 DB 事务提交成功后再写 ws_events
   - 广播失败不能影响主流程（写库成功后，广播失败记录日志即可）

3. 实现 internal/ws/publisher.go：
   - PublishWebsiteEvent(eventType, payload) 方法
   - 统一写库 + 广播

## Phase 6: 编写验收测试（15+条 SQL + 12+条验证）

### 输入
- 完整实现的 Socket.IO 服务端

### 输出
- scripts/test_websocket.sh（验收测试脚本）

### 任务分解
1. 编写 SQL 验证（至少 15 条）
   - ws_events 写入
   - 按 lastEventId 拉取
   - topic 索引
   - event_type 过滤
   - payload 格式验证

2. 编写验证步骤（至少 12 条，curl + node 脚本皆可）
   - 无 token 连接被拒绝
   - 正常 token 连接成功收到 connected
   - request:websites 收到 websites:initial
   - create website 后收到 websites:update add
   - update website 后收到 websites:update update
   - delete website 后收到 websites:update delete
   - lastEventId=旧值 → 收到增量事件
   - lastEventId 太旧/事件过多 → 回落收到 websites:initial

## Phase 7: 生成交付报告并提交代码

### 输入
- 完整实现的功能
- 验收测试结果

### 输出
- docs/T2-09-DELIVERY.md（交付报告）
- Git commit 并推送到 GitHub

### 任务分解
1. 生成交付报告
   - 改动文件清单
   - go test ./... 结果
   - 至少 15 条 SQL 验证
   - 至少 12 条验证步骤
   - 回滚策略

2. 提交代码
   - git add -A
   - git commit -m "feat(T2-09): implement websocket real-time sync for websites"
   - git push origin main

## 技术方案

### Socket.IO 方案选择

方案 B：默认路径
- 使用 Socket.IO 默认路径：/socket.io/
- 监听地址复用现有 HTTP server（Gin 同端口）
- 连接方式（前端参考）：io("https://your-domain", { auth: { token } })

### JWT 认证方案

方案 A：统一方案
- 客户端通过 auth.token 传 JWT
- 服务端在 connection 阶段验证 JWT
- 验证失败：拒绝连接（断开）
- 复用现有 internal/auth/jwt.go 的校验函数
- 不使用 cookie session

### 事件模型

新增表：ws_events

字段要求（最小集）：
- id bigint pk auto_increment（作为 eventId）
- topic varchar(64) not null（固定填 "websites"）
- event_type enum('add','update','delete') not null
- payload json not null（推送内容，前端可直接用）
- created_at datetime

索引：
- index(topic, id)

### Socket.IO 事件定义

(1) 连接确认
- event：connected
- data：{ "ok": true }

(2) 初始全量
- 客户端发送：request:websites，payload 可为空或 { lastEventId?: number }
- 服务端响应：websites:initial
- data：
  ```json
  {
    "items": [ ...website_list_items... ],
    "total": 123,
    "version": 456789,
    "lastEventId": 1024
  }
  ```

(3) 增量更新广播
- event：websites:update
- data：
  ```json
  {
    "eventId": 1025,
    "type": "add|update|delete",
    "data": { ... }
  }
  ```

### 增量补发逻辑

当客户端发送：request:websites + { lastEventId: 1000 }

服务端逻辑：
- 查 ws_events where topic='websites' and id > lastEventId order by id asc limit 500
- 若查到的事件数量 <= 500：逐条 emit websites:update（按顺序）
- 若事件过多/超过 500：直接返回一次 websites:initial

注意：P0 不要求"严格连续"，只要求：
- 有增量就推增量
- 增量过多就回全量兜底

### CRUD 联动

在以下 HTTP API 成功提交事务后写入事件表并广播：
- POST /api/v1/websites/create → add
- POST /api/v1/websites/update → update
- POST /api/v1/websites/delete → delete

要求：
- 必须在 DB 事务提交成功后再写 ws_events
- 广播失败不能影响主流程（写库成功后，广播失败记录日志即可）

## P1 可选功能（建议同任务完成）

P1-1) 连接房间/命名空间隔离
- 把网站订阅加入 room：room = "websites"
- 广播只对该 room
- （后续加 domains/nodes 等 topic 更容易）

P1-2) 事件清理策略
- 定期清理 ws_events（例如保留 7 天或最大 100 万条）
- 可用 cron/worker，或仅提供手工 SQL 文档即可

## 禁止项

- 禁止写前端
- 禁止出现 /ws 自定义路径（方案 B：默认 /socket.io）
- 禁止使用 cookie session
- 禁止调用 Agent
- 禁止使用图标（emoji）

## 验收标准

- 前端只要连上 Socket.IO 就能拿到全量并接收增量
- JWT 认证统一且可靠
- CRUD 变更能实时广播
- lastEventId 机制可用（增量或全量兜底）
- go test ./... 通过
- 至少 15 条 SQL 验证通过
- 至少 12 条验证步骤通过

## 回滚策略

- 关闭 Socket.IO 挂载（注释路由/开关 ENV）
- drop ws_events 表（如需彻底回退）
- revert commit

## 相关文档

- T2-09 任务需求：/home/ubuntu/upload/pasted_content_20.txt
- 现有 JWT 认证：internal/auth/jwt.go
- 现有 websites CRUD：api/v1/websites/handler.go
