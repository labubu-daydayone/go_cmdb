# T2-03改造清单v1.0 - 交付摘要

## 提交信息

**提交哈希**: `9e052f2`  
**GitHub**: https://github.com/labubu-daydayone/go_cmdb/commit/9e052f2  
**提交日期**: 2026-01-23

---

## 改造清单

| 改造项 | 优先级 | 状态 | 说明 |
|-------|-------|------|------|
| **P0-01**: version唯一性 | P0 | ✅ | 数据库递增，version=id |
| **P0-02**: Agent回滚能力 | P0 | ✅ | 保留versions/目录，live symlink |
| **P0-03**: nginx -t错误详情 | P0 | ✅ | 记录cmd、exitCode、stderr |
| **P1-01**: redirect策略 | P1 | ✅ | 不生成upstream文件（已实现） |
| **P1-02**: 状态回写 | P1 | ✅ | config_versions状态同步 |
| **P1-03**: acme策略 | P1 | ✅ | 未签发自动降级HTTP（已实现） |

---

## 文件变更

### 新增文件（1个）
- `scripts/test_refactor_v1.sh` - 验收测试脚本

### 修改文件（9个）
- `internal/model/config_version.go` - 添加Reason和LastError字段
- `internal/configver/service.go` - CreateVersion数据库递增
- `api/v1/config/handler.go` - Apply流程调整
- `agent/config/dirs.go` - versions目录+live symlink
- `agent/executor/apply_config.go` - 切换逻辑+nginx错误详情
- `internal/agent/dispatcher.go` - 状态回写逻辑
- `migrations/006_create_config_versions.sql` - 添加reason和last_error字段
- `docs/T2-03-DELIVERY.md` - T2-03初始交付报告
- `docs/T2-03-SUMMARY.md` - T2-03初始交付摘要

---

## 核心改进

### P0-01: version唯一性
```
时间戳（并发可能重复）→ 数据库auto_increment ID（天然唯一）
```

### P0-02: Agent回滚能力
```
/etc/nginx/cmdb/
├── versions/
│   ├── 1/  (保留历史版本)
│   ├── 2/  (保留历史版本)
│   └── ...
└── live -> versions/2/  (symlink指向当前版本)
```

### P0-03: nginx -t错误详情
```json
{
  "version": 123,
  "cmd": "nginx -t -c /etc/nginx/nginx.conf",
  "exitCode": 1,
  "stderr": "nginx: [emerg] invalid server_name...",
  "error": "nginx test failed (exit code 1): ...",
  "time": "2026-01-23T12:00:00Z"
}
```

### P1-02: 状态回写
```
pending（创建）→ applied（成功）
pending（创建）→ failed（失败，记录last_error）
```

---

## 验收要点

### 1. 并发测试（P0-01）
```bash
# 并发50次apply
for i in {1..50}; do curl -X POST /api/v1/config/apply -d '{"nodeId":1}' & done; wait

# SQL验证：无重复version
SELECT version, COUNT(*) FROM config_versions GROUP BY version HAVING COUNT(*) > 1;
-- 预期: Empty set
```

### 2. 回滚测试（P0-02）
```bash
# 两次apply
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'  # version=1
curl -X POST /api/v1/config/apply -d '{"nodeId":1}'  # version=2

# Agent端验证
ls /etc/nginx/cmdb/versions/  # 预期: 1/ 2/
ls -la /etc/nginx/cmdb/live   # 预期: live -> versions/2/
```

### 3. 错误详情测试（P0-03）
```bash
# 触发nginx -t失败（需手动创建无效配置）

# Agent端验证
cat /etc/nginx/cmdb/meta/last_error.json
# 预期: 包含cmd、exitCode、stderr、error、time字段
```

### 4. 状态回写测试（P1-02）
```sql
-- 统计各状态的版本数量
SELECT status, COUNT(*) FROM config_versions GROUP BY status;
-- 预期: pending/applied/failed三种状态
```

---

## 验收SQL（10条）

```sql
-- 1. 检查version无重复
SELECT version, COUNT(*) FROM config_versions GROUP BY version HAVING COUNT(*) > 1;

-- 2. 检查version等于id
SELECT id, version FROM config_versions WHERE id != version LIMIT 10;

-- 3. 检查两个版本是否都存在
SELECT version, status FROM config_versions WHERE version IN (1, 2);

-- 4. 检查失败版本的last_error字段
SELECT version, status, last_error FROM config_versions WHERE status = 'failed' LIMIT 5;

-- 5. 统计各状态的版本数量
SELECT status, COUNT(*) FROM config_versions GROUP BY status;

-- 6. 查询失败版本的详细信息
SELECT version, status, last_error, created_at FROM config_versions WHERE status = 'failed' ORDER BY version DESC LIMIT 5;

-- 7. 查询最近10个版本的完整信息
SELECT id, version, node_id, reason, status, last_error, applied_at, created_at FROM config_versions ORDER BY version DESC LIMIT 10;

-- 8. 统计任务状态分布
SELECT status, COUNT(*) FROM agent_tasks WHERE type = 'apply_config' GROUP BY status;

-- 9. 关联查询config_versions和agent_tasks
SELECT cv.version, cv.status, at.status, at.last_error FROM config_versions cv LEFT JOIN agent_tasks at ON JSON_EXTRACT(at.payload, '$.version') = cv.version WHERE cv.node_id = 1 ORDER BY cv.version DESC LIMIT 10;

-- 10. 查询version递增情况
SELECT MIN(version) as min_version, MAX(version) as max_version, COUNT(*) as total_count, MAX(version) - MIN(version) + 1 as expected_count FROM config_versions;
```

---

## 回滚策略

### 代码回滚
```bash
git revert 9e052f2 && git push origin main
```

### 数据库回滚
```sql
ALTER TABLE config_versions DROP COLUMN reason;
ALTER TABLE config_versions DROP COLUMN last_error;
```

### Agent回滚
```bash
pkill -f "bin/agent"
rm -rf /etc/nginx/cmdb/versions /etc/nginx/cmdb/live
mkdir -p /etc/nginx/cmdb/{upstreams,servers,certs,meta}
./bin/agent.old &
```

---

## 已知限制

1. version gap（CreateVersion成功但后续失败）
2. 历史版本需手动清理
3. last_error截断到255字符
4. stderr截断到2KB

---

## 统计数据

| 指标 | 数量 |
|-----|------|
| 新增文件 | 1 |
| 修改文件 | 9 |
| 改造项 | 6 |
| 测试场景 | 9 |
| SQL验证 | 10 |

---

## 交付完成

T2-03改造清单v1.0已完整交付，所有P0和P1改造均已完成。

**详细文档**: `docs/T2-03-REFACTOR-DELIVERY.md`  
**测试脚本**: `scripts/test_refactor_v1.sh`  
**GitHub**: https://github.com/labubu-daydayone/go_cmdb/commit/9e052f2
