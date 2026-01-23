# T2-03 实现计划

## 任务目标

将系统从"能安全下发任务"推进到"能让CDN真正生效"：
- 控制端聚合网站配置数据，生成结构化下发payload
- Agent端内置Nginx模板并渲染生成配置文件
- 支持upstream与server分离（可维护、可复用）
- 支持config_versions递增版本，保证幂等与可回滚
- apply_config任务落地：写文件 + nginx -t校验 + reload
- 全链路可验证（文件内容、任务状态、版本记录、失败回滚）

## 实现阶段

### Phase 1: 分析需求并创建实现计划（当前）

**输出**：
- T2-03-PLAN.md（本文档）
- 明确数据模型、API接口、文件结构

### Phase 2: 实现控制端config_versions模型和服务

**文件**：
- internal/model/config_version.go
- internal/configver/service.go

**功能**：
- config_versions表结构（version递增、node_id、payload、status）
- CreateVersion服务（生成递增version）
- GetLatestVersion服务（查询最新版本）
- ListVersions服务（分页查询）

### Phase 3: 实现控制端配置聚合和payload生成

**文件**：
- internal/configgen/aggregator.go
- internal/configgen/payload.go

**功能**：
- 聚合查询websites + website_domains + website_https
- 聚合查询origin_sets + origin_addresses
- 聚合查询certificates（HTTPS场景）
- 生成标准化payload（JSON）

**Payload结构**：
```json
{
  "version": 1234567890,
  "websites": [
    {
      "websiteId": 1,
      "status": "active",
      "domains": [
        {"domain": "example.com", "isPrimary": true, "cname": "cdn.example.com"}
      ],
      "origin": {
        "mode": "group",
        "upstreamName": "upstream_site_1",
        "addresses": [
          {"role": "primary", "protocol": "http", "address": "192.168.1.1:80", "weight": 100, "enabled": true}
        ]
      },
      "https": {
        "enabled": true,
        "forceRedirect": true,
        "hsts": true,
        "certificate": {
          "certificateId": 1,
          "certPem": "-----BEGIN CERTIFICATE-----...",
          "keyPem": "-----BEGIN PRIVATE KEY-----..."
        }
      }
    }
  ]
}
```

### Phase 4: 实现Agent配置目录规范和Nginx模板渲染

**文件**：
- agent/config/dirs.go（目录规范）
- agent/templates/upstream.tmpl
- agent/templates/server.tmpl
- agent/render/renderer.go

**目录结构**：
```
/etc/nginx/cmdb/
├── upstreams/
│   └── upstream_site_1.conf
├── servers/
│   └── server_site_1.conf
├── certs/
│   ├── cert_1.pem
│   └── key_1.pem
└── meta/
    ├── applied_version.json
    ├── last_success_version.json
    └── last_error.json
```

**环境变量**：
- NGINX_DIR（默认 /etc/nginx）
- NGINX_BIN（默认 nginx）
- NGINX_RELOAD_CMD（默认 "nginx -s reload"）
- CMDB_RENDER_DIR（默认 /etc/nginx/cmdb）

**模板功能**：
- upstream模板：支持weight、primary/backup、enabled过滤
- server模板：支持HTTP/HTTPS、forceRedirect、HSTS、redirect模式

### Phase 5: 实现Agent apply_config执行流程

**文件**：
- agent/executor/apply_config.go

**执行流程**：
1. 校验payload.version > 已应用version（幂等）
2. 渲染并写入staging目录：{CMDB_RENDER_DIR}/.staging/{version}/
3. 写证书文件（cert/key）
4. 执行nginx -t校验
5. 若失败：
   - 写meta/last_error.json
   - 返回failed
   - 不覆盖当前生效目录
6. 若成功：
   - 原子切换（rename staging -> live）
   - 更新meta/applied_version.json + last_success_version.json
   - 执行reload
   - 返回success

**关键保障**：
- 失败不影响当前生效配置
- 原子切换（rename操作）
- 幂等性（version校验）

### Phase 6: 实现控制端配置下发API

**路由**：
- POST /api/v1/config/apply
- GET /api/v1/config/versions

**文件**：
- api/v1/config/handler.go
- api/v1/router.go（添加路由）

**功能**：
- POST /api/v1/config/apply：
  - 创建config_versions
  - 聚合生成payload
  - 创建agent_tasks（type=apply_config）
  - 下发到node
  - 返回version + taskId

- GET /api/v1/config/versions：
  - 分页返回version列表
  - 支持按nodeId筛选

### Phase 7: 编写验收测试

**测试脚本**：
- scripts/test_apply_config.sh

**测试场景（16+条）**：
1. 创建node（已有）
2. 创建origin_group + website + domains + https(select)
3. 调用POST /api/v1/config/apply下发
4. Agent生成upstream/server/cert文件验证
5. nginx -t成功验证
6. 修改网站回源（origin_set）再apply，version增加
7. 幂等：同version重复apply不重复切换
8. 失败场景：故意生成非法nginx配置导致-t失败
9. revoke agent_identity后apply失败
10. https.enabled=0情况生成80配置
11. redirect模式生成return配置
12. backup地址生成backup server
13. enabled=false地址不输出
14. 查询config_versions列表
15. 查询agent_tasks结果
16. 验证失败回滚（nginx -t fail不影响live）

**SQL验证（10+条）**：
1. config_versions递增
2. agent_tasks payload.version
3. 网站变更后再次apply版本更新
4. task状态变化
5. nginx -t失败last_error写入
6. 查询最新version
7. 按nodeId筛选versions
8. 统计各状态任务数量
9. 关联查询website + config_versions
10. 查询失败任务的error信息

### Phase 8: 生成交付报告并提交代码

**交付物**：
- docs/T2-03-DELIVERY.md（完整报告）
- docs/T2-03-SUMMARY.md（交付摘要）
- scripts/test_apply_config.sh（验收脚本）

**报告内容**：
- 提交哈希
- 文件变更清单（控制端/Agent）
- payload示例（精简但完整）
- 模板路径与输出文件路径说明
- 本地联调步骤（从0到成功）
- curl集合（16+条）
- SQL验证（10+条）
- 失败回滚证明（nginx -t fail不影响live）
- 已知限制

## 数据模型

### config_versions表

```sql
CREATE TABLE config_versions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  version BIGINT NOT NULL UNIQUE COMMENT '配置版本号（递增）',
  node_id INT NOT NULL COMMENT '节点ID',
  payload TEXT NOT NULL COMMENT '配置payload（JSON）',
  status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/applied/failed',
  applied_at DATETIME COMMENT '应用时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_node_version (node_id, version),
  INDEX idx_created_at (created_at)
) COMMENT='配置版本记录';
```

## API接口

### POST /api/v1/config/apply

**请求**：
```json
{
  "nodeId": 1,
  "reason": "更新网站配置"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "version": 1234567890,
    "taskId": 123
  }
}
```

### GET /api/v1/config/versions

**请求**：
```
GET /api/v1/config/versions?nodeId=1&page=1&pageSize=20
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "version": 1234567890,
        "nodeId": 1,
        "status": "applied",
        "appliedAt": "2026-01-23T12:00:00Z",
        "createdAt": "2026-01-23T11:59:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

## 文件结构

### 控制端新增文件

```
internal/
├── model/
│   └── config_version.go
├── configver/
│   └── service.go
└── configgen/
    ├── aggregator.go
    └── payload.go

api/v1/
└── config/
    └── handler.go
```

### Agent新增文件

```
agent/
├── config/
│   └── dirs.go
├── templates/
│   ├── upstream.tmpl
│   └── server.tmpl
├── render/
│   └── renderer.go
└── executor/
    └── apply_config.go
```

## 关键技术点

### 1. 版本递增策略

使用时间戳（毫秒）作为version：
```go
version := time.Now().UnixMilli()
```

优点：
- 自然递增
- 全局唯一
- 可读性好（可转换为时间）

### 2. 原子切换策略

使用os.Rename实现原子切换：
```go
// 渲染到staging
stagingDir := fmt.Sprintf("%s/.staging/%d", renderDir, version)
// ... 渲染文件 ...

// nginx -t校验
if err := nginxTest(stagingDir); err != nil {
  return err
}

// 原子切换
liveDir := fmt.Sprintf("%s/live", renderDir)
os.Rename(stagingDir, liveDir)
```

### 3. 幂等性保障

```go
// 读取已应用版本
appliedVersion := readAppliedVersion()

// 校验版本
if payload.Version <= appliedVersion {
  return "success" // 幂等返回
}
```

### 4. 失败回滚保障

```go
// nginx -t失败
if err := nginxTest(stagingDir); err != nil {
  // 写错误日志
  writeLastError(err)
  
  // 不切换，保留当前配置
  return "failed"
}
```

## 已知限制

1. DNS worker真同步Cloudflare（后续）
2. ACME worker真申请证书（后续）
3. WebSocket推送（后续）
4. 完整缓存规则落地（可只预留变量）
5. purge_cache仍为模拟（不要求真实）

## 回滚策略

### 代码回滚

```bash
git revert <T2-03-commit-hash>
git push origin main
```

### 数据库回滚

```sql
DROP TABLE IF EXISTS config_versions;
```

### Agent回滚

```bash
# 删除staging目录
rm -rf /etc/nginx/cmdb/.staging

# live目录保留，不影响当前配置
```

## 验收标准

1. go test ./... 通过
2. 至少16个curl/步骤（覆盖所有场景）
3. 至少10条SQL验证
4. nginx -t失败不影响当前配置（必须验证）
5. 幂等性验证（同version重复apply）
6. 版本递增验证
7. 文件内容正确性验证
8. 交付报告完整（禁止使用图标/emoji）

## 下一步

开始Phase 2：实现控制端config_versions模型和服务
