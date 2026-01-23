# T2-03改造清单v1.0 - 完整交付报告

## 提交信息

- **提交哈希**: `9e052f2`
- **GitHub仓库**: https://github.com/labubu-daydayone/go_cmdb
- **提交日期**: 2026-01-23
- **提交信息**: refactor(T2-03): 改造清单v1.0 - version唯一性+Agent回滚+nginx错误详情+状态回写

---

## 改造概述

对T2-03进行6项改造，提升系统稳定性、可维护性和可追溯性。

---

## 文件变更清单

### 新增文件（1个）

- `scripts/test_refactor_v1.sh` - 验收测试脚本

### 修改文件（9个）

| 文件 | 改造项 | 说明 |
|-----|-------|------|
| `internal/model/config_version.go` | P0-01, P1-02 | 添加Reason和LastError字段 |
| `internal/configver/service.go` | P0-01, P1-02 | CreateVersion改为数据库递增，UpdateStatus支持lastError |
| `api/v1/config/handler.go` | P0-01 | Apply流程改为先创建version再生成payload |
| `agent/config/dirs.go` | P0-02, P0-03 | 添加versions目录管理、live symlink、NginxTestCmd配置 |
| `agent/executor/apply_config.go` | P0-02, P0-03 | 改造切换逻辑、nginx -t错误详情、ErrorMeta结构 |
| `internal/agent/dispatcher.go` | P1-02 | 添加config_versions状态回写逻辑 |
| `migrations/006_create_config_versions.sql` | P0-01, P1-02 | 添加reason和last_error字段 |
| `docs/T2-03-DELIVERY.md` | - | T2-03初始交付报告（已存在） |
| `docs/T2-03-SUMMARY.md` | - | T2-03初始交付摘要（已存在） |

---

## 改造详情

### P0-01: version生成改为数据库递增（全局唯一）

**问题**: 原使用时间戳毫秒作为version，并发场景下可能重复

**改造**:
1. CreateVersion方法改造（internal/configver/service.go）
   - Step 1: 插入记录（version=0占位）获取auto_increment ID
   - Step 2: 更新version=ID（保证全局唯一）
   - Step 3: 返回configVersion（version已等于ID）

2. Apply API流程调整（api/v1/config/handler.go）
   - Step 1: 先创建config_version获取ID
   - Step 2: 使用version ID生成payload
   - Step 3: 更新payload到config_version
   - Step 4: 创建agent_task
   - Step 5: 异步下发任务

3. 数据库迁移SQL更新（migrations/006_create_config_versions.sql）
   - 添加reason字段（varchar(255)）
   - 添加last_error字段（varchar(255)）

**验收**:
```bash
# 并发50次apply
for i in {1..50}; do
  curl -X POST /api/v1/config/apply -d '{"nodeId":1}' &
done
wait

# SQL验证：检查version无重复
SELECT version, COUNT(*) as cnt 
FROM config_versions 
GROUP BY version 
HAVING cnt > 1;
-- 预期结果: Empty set

# SQL验证：检查version等于id
SELECT id, version 
FROM config_versions 
WHERE id != version 
LIMIT 10;
-- 预期结果: Empty set
```

---

### P0-02: Agent回滚能力固化（保留历史版本）

**问题**: 原staging直接rename到live，无法保留历史版本

**改造**:
1. 目录规范改造（agent/config/dirs.go）
   - 添加GetVersionsDir()：返回versions目录路径
   - 添加GetVersionDir(version)：返回特定版本目录路径
   - 添加AtomicSwitchToVersion(version)：原子切换live symlink
   - 修改EnsureDirectories()：只创建meta和versions目录

2. 切换逻辑改造（agent/executor/apply_config.go）
   - Step 5: 移动staging到versions/{version}/
   - Step 6: 原子切换live symlink指向新版本
   - Step 7: 更新元数据
   - Step 8: reload nginx

**目录结构**:
```
/etc/nginx/cmdb/
├── versions/
│   ├── 1/
│   │   ├── upstreams/
│   │   ├── servers/
│   │   └── certs/
│   ├── 2/
│   │   ├── upstreams/
│   │   ├── servers/
│   │   └── certs/
│   └── ...
├── live -> versions/2/  (symlink指向最新版本)
├── meta/
│   ├── applied_version.json
│   ├── last_success_version.json
│   └── last_error.json
└── .staging/
    └── {version}/  (临时目录)
```

**验收**:
```bash
# 第一次apply
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'
# version=1

# 第二次apply
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'
# version=2

# Agent端验证
ls -la /etc/nginx/cmdb/versions/
# 预期: 存在1/和2/两个目录

ls -la /etc/nginx/cmdb/live
# 预期: live -> versions/2/

# 手动回滚到version 1
ln -sfn /etc/nginx/cmdb/versions/1 /etc/nginx/cmdb/live
nginx -s reload
```

---

### P0-03: nginx -t错误详情规范（记录完整stderr）

**问题**: 原last_error.json只记录简单错误信息，无法定位问题

**改造**:
1. DirConfig配置增强（agent/config/dirs.go）
   - 添加NginxConf字段（默认/etc/nginx/nginx.conf）
   - 添加NginxTestCmd字段（默认nginx -t -c /etc/nginx/nginx.conf）

2. ErrorMeta结构增强（agent/executor/apply_config.go）
   - 添加Cmd字段（记录执行的命令）
   - 添加ExitCode字段（记录退出码）
   - 添加Stderr字段（记录完整stderr，截断到2KB）

3. NginxTestError错误类型（agent/executor/apply_config.go）
   - 新增NginxTestError结构体
   - 包含Cmd、ExitCode、Stderr三个字段

4. nginxTest方法改造（agent/executor/apply_config.go）
   - 使用NginxTestCmd配置执行测试
   - 捕获完整stderr输出
   - 提取exit code
   - 截断stderr到2KB（防止过大）
   - 返回NginxTestError类型

5. writeLastError方法改造（agent/executor/apply_config.go）
   - 检测NginxTestError类型
   - 记录cmd、exitCode、stderr到last_error.json

**last_error.json示例**:
```json
{
  "version": 123,
  "cmd": "nginx -t -c /etc/nginx/nginx.conf",
  "exitCode": 1,
  "stderr": "nginx: [emerg] invalid server_name \"\" in /etc/nginx/cmdb/live/servers/server_site_1.conf:3\nnginx: configuration file /etc/nginx/nginx.conf test failed",
  "error": "nginx test failed (exit code 1): nginx: [emerg] invalid server_name...",
  "time": "2026-01-23T12:00:00Z"
}
```

**验收**:
```bash
# 创建无效配置触发nginx -t失败
# （需要手动修改数据库或通过API创建无效域名）

# Agent端验证
cat /etc/nginx/cmdb/meta/last_error.json
# 预期: 包含cmd、exitCode、stderr、error、time字段

# SQL验证
SELECT version, status, last_error 
FROM config_versions 
WHERE status = 'failed' 
LIMIT 5;
-- 预期: last_error字段包含错误详情
```

---

### P1-01: redirect模式策略固定（不生成upstream文件）

**状态**: 已在T2-03初始实现中完成

**验证**:
```go
// agent/executor/apply_config.go:182
if website.Origin.Mode != "redirect" {
    // 渲染upstream配置
}
```

**验收**:
```bash
# 创建redirect模式网站
curl -X POST /api/v1/websites -d '{
  "name": "redirect_test",
  "originMode": "redirect",
  "redirectUrl": "https://example.com"
}'

# apply配置
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'

# Agent端验证
ls /etc/nginx/cmdb/live/upstreams/ | grep upstream_site_{id}
# 预期: 无输出（不应生成upstream文件）

cat /etc/nginx/cmdb/live/servers/server_site_{id}.conf
# 预期: 包含 'return 301 https://example.com;' 语句
```

---

### P1-02: config_versions状态回写（created/applied/failed）

**问题**: 原config_versions状态不更新，无法追踪配置应用结果

**改造**:
1. UpdateStatus方法增强（internal/configver/service.go）
   - 添加可选参数lastError（变长参数）
   - status=failed时更新last_error字段
   - status=applied时更新applied_at字段

2. Dispatcher状态回写（internal/agent/dispatcher.go）
   - 添加updateConfigVersionStatus辅助方法
   - 从task payload中提取version
   - 截断last_error到255字符（数据库字段限制）
   - 在4个失败场景下回写failed状态
   - 在成功场景下回写applied状态

**状态流转**:
```
pending（创建时）→ applied（成功）
pending（创建时）→ failed（失败）
```

**验收**:
```bash
# 查询config_versions状态
curl -X GET '/api/v1/config/versions?nodeId=1&page=1&pageSize=10'

# SQL验证：统计各状态的版本数量
SELECT status, COUNT(*) as cnt 
FROM config_versions 
GROUP BY status;
-- 预期结果: pending/applied/failed三种状态

# SQL验证：查询失败版本的详细信息
SELECT version, status, last_error, created_at 
FROM config_versions 
WHERE status = 'failed' 
ORDER BY version DESC 
LIMIT 5;
-- 预期结果: last_error字段有内容

# SQL验证：关联查询config_versions和agent_tasks
SELECT 
  cv.version, 
  cv.status as cv_status, 
  at.status as task_status, 
  at.last_error 
FROM config_versions cv 
LEFT JOIN agent_tasks at ON JSON_EXTRACT(at.payload, '$.version') = cv.version 
WHERE cv.node_id = 1 
ORDER BY cv.version DESC 
LIMIT 10;
```

---

### P1-03: acme未签发策略固化（自动降级为HTTP）

**状态**: 已在T2-03初始实现中完成

**验证**:
```go
// internal/configgen/aggregator.go:195-218
} else if websiteHTTPS.CertMode == model.CertModeACME {
    if websiteHTTPS.CertificateID == 0 {
        config.Enabled = false  // 未关联证书，降级为HTTP
        return config, nil
    }

    var certificate model.Certificate
    if err := a.db.First(&certificate, websiteHTTPS.CertificateID).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            config.Enabled = false  // 证书不存在，降级为HTTP
            return config, nil
        }
        return nil, fmt.Errorf("failed to query ACME certificate: %w", err)
    }

    if certificate.Status != model.CertificateStatusIssued {
        config.Enabled = false  // 证书未签发，降级为HTTP
        return config, nil
    }

    config.Certificate = &CertificateConfig{...}
}
```

**验收**:
```bash
# 创建ACME模式网站（未签发证书）
curl -X POST /api/v1/websites -d '{
  "name": "acme_test",
  "httpsEnabled": true,
  "certMode": "acme",
  "certificateId": 0
}'

# apply配置
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'

# Agent端验证
cat /etc/nginx/cmdb/live/servers/server_site_{id}.conf | grep 'listen 443'
# 预期: 无输出（应降级为HTTP，不监听443）

cat /etc/nginx/cmdb/live/servers/server_site_{id}.conf | grep 'listen 80'
# 预期: 有输出（应监听80端口）
```

---

## 验收测试脚本

### 使用方法

```bash
# 1. 修改脚本中的BASE_URL和ADMIN_TOKEN
vi scripts/test_refactor_v1.sh

# 2. 执行测试脚本
./scripts/test_refactor_v1.sh

# 3. 在Agent服务器执行验证命令
tree -L 3 /etc/nginx/cmdb/
cat /etc/nginx/cmdb/meta/last_error.json
ls -la /etc/nginx/cmdb/live

# 4. 在数据库执行SQL验证查询
# （脚本中包含10条SQL验证查询）
```

### 测试覆盖

| 测试项 | 测试场景 | 验证方法 |
|-------|---------|---------|
| P0-01 | 并发50次apply | SQL检查version无重复 |
| P0-01 | version等于id | SQL检查id=version |
| P0-02 | 两次apply | Agent端检查versions/目录 |
| P0-02 | live symlink | Agent端检查live指向 |
| P0-03 | nginx -t失败 | Agent端检查last_error.json |
| P0-03 | 错误详情 | SQL检查last_error字段 |
| P1-01 | redirect模式 | Agent端检查无upstream文件 |
| P1-02 | 状态回写 | SQL统计status分布 |
| P1-03 | acme未签发 | Agent端检查降级为HTTP |

---

## 综合验收SQL

```sql
-- SQL验证1: 检查version无重复
SELECT version, COUNT(*) as cnt 
FROM config_versions 
GROUP BY version 
HAVING cnt > 1;
-- 预期结果: Empty set

-- SQL验证2: 检查version等于id
SELECT id, version 
FROM config_versions 
WHERE id != version 
LIMIT 10;
-- 预期结果: Empty set

-- SQL验证3: 检查两个版本是否都存在
SELECT version, status 
FROM config_versions 
WHERE version IN (1, 2);
-- 预期结果: 两条记录

-- SQL验证4: 检查失败版本的last_error字段
SELECT version, status, last_error 
FROM config_versions 
WHERE status = 'failed' 
LIMIT 5;
-- 预期结果: last_error字段包含错误详情

-- SQL验证5: 统计各状态的版本数量
SELECT status, COUNT(*) as cnt 
FROM config_versions 
GROUP BY status;
-- 预期结果: pending/applied/failed三种状态

-- SQL验证6: 查询失败版本的详细信息
SELECT version, status, last_error, created_at 
FROM config_versions 
WHERE status = 'failed' 
ORDER BY version DESC 
LIMIT 5;
-- 预期结果: last_error字段有内容

-- SQL验证7: 查询最近10个版本的完整信息
SELECT id, version, node_id, reason, status, last_error, applied_at, created_at 
FROM config_versions 
ORDER BY version DESC 
LIMIT 10;

-- SQL验证8: 统计任务状态分布
SELECT status, COUNT(*) as cnt 
FROM agent_tasks 
WHERE type = 'apply_config' 
GROUP BY status;

-- SQL验证9: 关联查询config_versions和agent_tasks
SELECT 
  cv.version, 
  cv.status as cv_status, 
  at.status as task_status, 
  at.last_error 
FROM config_versions cv 
LEFT JOIN agent_tasks at ON JSON_EXTRACT(at.payload, '$.version') = cv.version 
WHERE cv.node_id = 1 
ORDER BY cv.version DESC 
LIMIT 10;

-- SQL验证10: 查询version递增情况
SELECT 
  MIN(version) as min_version, 
  MAX(version) as max_version, 
  COUNT(*) as total_count, 
  MAX(version) - MIN(version) + 1 as expected_count 
FROM config_versions;
-- 预期结果: total_count 接近 expected_count（允许有少量gap）
```

---

## 回滚策略

### 代码回滚

```bash
# Git回滚到上一个版本
git revert 9e052f2
git push origin main

# 或者硬回滚（慎用）
git reset --hard 37ee21b
git push -f origin main
```

### 数据库回滚

```sql
-- 删除reason和last_error字段（如果需要）
ALTER TABLE config_versions DROP COLUMN reason;
ALTER TABLE config_versions DROP COLUMN last_error;

-- 或者重建表（会丢失数据）
DROP TABLE IF EXISTS config_versions;
-- 然后运行旧的迁移SQL
```

### Agent回滚

```bash
# 停止新版Agent
pkill -f "bin/agent"

# 清理新版目录结构
rm -rf /etc/nginx/cmdb/versions
rm -f /etc/nginx/cmdb/live

# 恢复旧版目录结构
mkdir -p /etc/nginx/cmdb/{upstreams,servers,certs,meta}

# 启动旧版Agent
./bin/agent.old &
```

---

## 已知限制

1. **version gap**: 如果CreateVersion成功但后续步骤失败，会产生version gap（id不连续）
2. **历史版本清理**: 当前未实现自动清理旧版本，需要手动清理`/etc/nginx/cmdb/versions/`
3. **并发限制**: 虽然version唯一，但高并发下可能导致数据库连接池耗尽
4. **last_error截断**: last_error字段限制255字符，超长错误会被截断
5. **nginx -t stderr截断**: stderr限制2KB，超长输出会被截断

---

## 统计数据

| 指标 | 数量 |
|-----|------|
| 新增文件 | 1 |
| 修改文件 | 9 |
| 新增代码行 | ~400 |
| 改造项 | 6 |
| P0改造项 | 3 |
| P1改造项 | 3 |
| 测试场景 | 9 |
| SQL验证 | 10 |

---

## 验收清单

- [x] P0-01: version生成改为数据库递增（全局唯一）
- [x] P0-02: Agent回滚能力固化（保留历史版本）
- [x] P0-03: nginx -t错误详情规范（记录完整stderr）
- [x] P1-01: redirect模式策略固定（不生成upstream文件）
- [x] P1-02: config_versions状态回写（created/applied/failed）
- [x] P1-03: acme未签发策略固化（自动降级为HTTP）
- [x] 验收测试脚本编写
- [x] 代码编译通过
- [x] 代码提交到GitHub
- [x] 交付报告完整

---

## 交付完成

T2-03改造清单v1.0已完整交付，所有P0和P1改造均已完成，可以进行验收。

**交付人**: Manus AI  
**交付日期**: 2026-01-23  
**GitHub**: https://github.com/labubu-daydayone/go_cmdb/commit/9e052f2
