# T2-05-fix 完整交付报告

ACME 自动 apply_config + retry 语义修正 + failed challenge 清理

---

## 1. 基本信息

- 仓库地址: https://github.com/labubu-daydayone/go_cmdb
- 提交哈希: 522b45b
- 交付日期: 2026-01-23
- 任务编号: T2-05-fix
- 优先级: P0

---

## 2. 变更文件清单

### 修改文件（2个）

1. internal/acme/service.go
   - 添加OnCertificateIssued方法（证书签发后自动apply_config）
   - 修改ResetRetry方法（保留attempts和last_error）
   - 修改MarkAsFailed方法（超限时自动清理challenge）
   - 添加CleanupFailedChallenge方法（清理TXT记录）

2. internal/acme/worker.go
   - 修改processRequest成功分支（调用OnCertificateIssued）
   - 移除旧的triggerWebsiteApply方法

### 新增文件（1个）

3. scripts/test_t2_05_fix.sh
   - 验收测试脚本（20条curl + 15条SQL）

---

## 3. ACME Worker 运行说明

### 启动方式

ACME Worker 在控制端主程序启动时自动启用：

```go
// cmd/cmdb/main.go
acmeWorker := acme.NewWorker(db, dnsService, acme.WorkerConfig{
    Enabled:     true,
    IntervalSec: 40,
    BatchSize:   100,
})
acmeWorker.Start()
```

### 环境变量列表

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| ACME_WORKER_ENABLED | true | 是否启用ACME Worker |
| ACME_WORKER_INTERVAL | 40 | 轮询间隔（秒） |
| ACME_WORKER_BATCH_SIZE | 100 | 批量处理大小 |

### 轮询间隔与批量大小

- 轮询间隔: 40秒（固定）
- 批量大小: 100条（每次处理最多100个证书请求）

---

## 4. 三个 P0 点逐条说明

### P0-01: 证书签发成功后自动 apply_config

**改动位置**: internal/acme/service.go

**实现方式**:

1. 添加OnCertificateIssued方法
   - 查询certificate_bindings（通过certificate_request_id）
   - 更新website_https.certificate_id = certID
   - 激活certificate_bindings（status = active）
   - 创建config_versions（reason = "acme-issued:{certID}"）
   - 创建agent_tasks（type = apply_config，全量下发）

2. 幂等性保证
   - 通过reason字段检查（acme-issued:{certID}）
   - 如果已存在相同reason的config_versions，跳过创建

3. Worker集成
   - 成功分支调用OnCertificateIssued
   - 证书复用分支也调用OnCertificateIssued

**验收点**:
- config_versions自动生成（reason = acme-issued:{certID}）
- agent_tasks自动生成（type = apply_config）
- website_https.certificate_id自动更新
- certificate_bindings自动激活（status = active）

---

### P0-02: retry API 语义修正（不清 attempts）

**改动位置**: internal/acme/service.go

**实现方式**:

修改ResetRetry方法：
- 仅限status = failed允许retry
- status: failed → pending
- attempts: 保持不变（不重置）
- last_error: 保留（不清空）
- updated_at: 更新为当前时间

**验收点**:
- retry后status变为pending
- attempts保持原值
- last_error保留
- Worker下次tick会继续处理

---

### P0-03: failed 超限自动清理 challenge TXT

**改动位置**: internal/acme/service.go

**实现方式**:

1. 添加CleanupFailedChallenge方法
   - 查找owner_type = acme_challenge且owner_id = requestID的TXT记录
   - 将desired_state从present改为absent
   - 幂等性保证（只更新present记录）

2. MarkAsFailed方法集成
   - 当attempts >= poll_max_attempts时触发cleanup
   - 失败不影响整体流程（仅记录日志）

3. DNS Worker后续处理
   - DNS Worker检测到desired_state = absent
   - 删除Cloudflare云端记录
   - 最终硬删除本地记录

**验收点**:
- failed超限后TXT记录desired_state变为absent
- DNS Worker删除云端记录
- 最终本地记录硬删除

---

## 5. API 验证（20 条 curl）

### Test 1: Create ACME provider
```bash
curl -X POST "http://localhost:8080/api/v1/acme/providers/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "letsencrypt",
    "directoryUrl": "https://acme-v02.api.letsencrypt.org/directory",
    "requiresEab": false
  }'
```

### Test 2: Create ACME account
```bash
curl -X POST "http://localhost:8080/api/v1/acme/accounts/create" \
  -H "Content-Type: application/json" \
  -d '{
    "providerName": "letsencrypt",
    "email": "admin@example.com"
  }'
```

### Test 3: Create domain
```bash
curl -X POST "http://localhost:8080/api/v1/domains/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example.com",
    "provider": "cloudflare",
    "apiToken": "test_token",
    "zoneId": "test_zone_id"
  }'
```

### Test 4: Create website with ACME mode
```bash
curl -X POST "http://localhost:8080/api/v1/websites/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Example Website",
    "domains": ["www.example.com"],
    "lineGroupId": 1,
    "httpsEnabled": true,
    "certMode": "acme"
  }'
```

### Test 5: Create certificate request
```bash
curl -X POST "http://localhost:8080/api/v1/acme/certificate/request" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": 1,
    "domains": ["www.example.com"],
    "websiteIds": [1]
  }'
```

### Test 6: Wait for ACME Worker (40 seconds)
```bash
sleep 40
```

### Test 7: Check certificate request status
```bash
curl -X GET "http://localhost:8080/api/v1/acme/certificate/requests/1"
```

### Test 8: List config_versions
```bash
curl -X GET "http://localhost:8080/api/v1/config/versions?page=1&pageSize=10"
```

### Test 9: List agent_tasks
```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks?page=1&pageSize=10"
```

### Test 10: Check website_https
```bash
curl -X GET "http://localhost:8080/api/v1/websites/1/https"
```

### Test 11: Create failing certificate request
```bash
curl -X POST "http://localhost:8080/api/v1/acme/certificate/request" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": 1,
    "domains": ["fail.example.com"],
    "websiteIds": []
  }'
```

### Test 12: Wait for failure (40 seconds)
```bash
sleep 40
```

### Test 13: Check failed request status
```bash
curl -X GET "http://localhost:8080/api/v1/acme/certificate/requests/2"
```

### Test 14: List DNS records (challenge)
```bash
curl -X GET "http://localhost:8080/api/v1/dns/records?ownerType=acme_challenge&page=1&pageSize=10"
```

### Test 15: Retry failed request
```bash
curl -X POST "http://localhost:8080/api/v1/acme/certificate/retry" \
  -H "Content-Type: application/json" \
  -d '{"id": 2}'
```

### Test 16: Check retry result
```bash
curl -X GET "http://localhost:8080/api/v1/acme/certificate/requests/2"
```

### Test 17: List certificate_bindings
```bash
curl -X GET "http://localhost:8080/api/v1/acme/certificate/bindings?requestId=1"
```

### Test 18: Check config_versions reason
```bash
curl -X GET "http://localhost:8080/api/v1/config/versions?page=1&pageSize=10"
```

### Test 19: Check agent_tasks payload
```bash
curl -X GET "http://localhost:8080/api/v1/agent-tasks/1"
```

### Test 20: List certificates
```bash
curl -X GET "http://localhost:8080/api/v1/certificates?page=1&pageSize=10"
```

---

## 6. SQL 验证（15 条 SQL）

### SQL Test 1: Verify config_versions with reason=acme-issued
```sql
SELECT id, version, node_id, status, reason, created_at 
FROM config_versions 
WHERE reason LIKE 'acme-issued:%' 
ORDER BY id DESC LIMIT 5;
```

### SQL Test 2: Verify agent_tasks with type=apply_config
```sql
SELECT id, node_id, type, payload, status, created_at 
FROM agent_tasks 
WHERE type='apply_config' 
ORDER BY id DESC LIMIT 5;
```

### SQL Test 3: Verify agent_tasks payload contains version
```sql
SELECT id, payload 
FROM agent_tasks 
WHERE type='apply_config' AND payload LIKE '%version%' 
ORDER BY id DESC LIMIT 3;
```

### SQL Test 4: Verify failed request with attempts and last_error
```sql
SELECT id, status, attempts, last_error, created_at 
FROM certificate_requests 
WHERE status='failed' OR attempts > 0 
ORDER BY attempts DESC LIMIT 5;
```

### SQL Test 5: Verify challenge TXT records desired_state=absent
```sql
SELECT id, domain_id, type, name, value, desired_state, owner_type, owner_id 
FROM domain_dns_records 
WHERE owner_type='acme_challenge' 
ORDER BY id DESC LIMIT 5;
```

### SQL Test 6: Verify retry preserves attempts
```sql
SELECT id, status, attempts, last_error, updated_at 
FROM certificate_requests 
WHERE id=2;
```

### SQL Test 7: Verify certificate_bindings status=active
```sql
SELECT id, certificate_request_id, website_id, certificate_id, status, created_at 
FROM certificate_bindings 
WHERE certificate_request_id=1 
ORDER BY id DESC LIMIT 5;
```

### SQL Test 8: Verify website_https.certificate_id updated
```sql
SELECT id, website_id, enabled, cert_mode, certificate_id, updated_at 
FROM website_https 
WHERE website_id=1;
```

### SQL Test 9: Verify config_versions status progression
```sql
SELECT id, version, status, reason, created_at, applied_at 
FROM config_versions 
WHERE reason LIKE 'acme-issued:%' 
ORDER BY id DESC LIMIT 5;
```

### SQL Test 10: Count challenge records by desired_state
```sql
SELECT desired_state, COUNT(*) as count 
FROM domain_dns_records 
WHERE owner_type='acme_challenge' 
GROUP BY desired_state;
```

### SQL Test 11: Verify certificate fingerprint uniqueness
```sql
SELECT fingerprint, COUNT(*) as count 
FROM certificates 
GROUP BY fingerprint 
HAVING count > 1;
```

### SQL Test 12: Verify certificate_domains for SAN
```sql
SELECT cd.certificate_id, cd.domain, cd.is_wildcard, c.status 
FROM certificate_domains cd 
JOIN certificates c ON cd.certificate_id = c.id 
ORDER BY cd.certificate_id DESC LIMIT 10;
```

### SQL Test 13: Verify attempts increment on failure
```sql
SELECT id, status, attempts, poll_max_attempts, last_error 
FROM certificate_requests 
WHERE status='failed' OR attempts > 0 
ORDER BY attempts DESC LIMIT 5;
```

### SQL Test 14: Verify config_versions idempotency
```sql
SELECT reason, COUNT(*) as count 
FROM config_versions 
WHERE reason LIKE 'acme-issued:%' 
GROUP BY reason 
HAVING count > 1;
```

### SQL Test 15: Verify challenge cleanup completeness
```sql
SELECT cr.id as request_id, cr.status, 
       COUNT(ddr.id) as challenge_count, 
       SUM(CASE WHEN ddr.desired_state='absent' THEN 1 ELSE 0 END) as absent_count 
FROM certificate_requests cr 
LEFT JOIN domain_dns_records ddr ON ddr.owner_type='acme_challenge' AND ddr.owner_id=cr.id 
WHERE cr.status='failed' 
GROUP BY cr.id 
ORDER BY cr.id DESC LIMIT 5;
```

---

## 7. 关键 payload 示例

### certificate_request 示例
```json
{
  "id": 1,
  "accountId": 1,
  "domains": "[\"www.example.com\"]",
  "status": "success",
  "attempts": 0,
  "pollMaxAttempts": 10,
  "lastError": "",
  "createdAt": "2026-01-23T12:00:00Z"
}
```

### domain_dns_records (TXT) 示例
```json
{
  "id": 1,
  "domainId": 1,
  "type": "TXT",
  "name": "_acme-challenge.www",
  "value": "random_challenge_token",
  "ttl": 120,
  "proxied": false,
  "status": "active",
  "desiredState": "present",
  "ownerType": "acme_challenge",
  "ownerId": 1,
  "providerRecordId": "cloudflare_record_id"
}
```

### apply_config 触发示例
```json
{
  "configVersion": {
    "id": 1,
    "version": 1,
    "nodeId": 0,
    "payload": "{...}",
    "status": "pending",
    "reason": "acme-issued:1"
  },
  "agentTask": {
    "id": 1,
    "nodeId": 1,
    "type": "apply_config",
    "payload": "{\"version\": 1}",
    "status": "pending"
  }
}
```

---

## 8. 失败与回滚证明

### 构造失败场景

使用无效的Cloudflare API token创建证书请求：

```bash
curl -X POST "http://localhost:8080/api/v1/acme/certificate/request" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": 1,
    "domains": ["fail.example.com"],
    "websiteIds": []
  }'
```

等待40秒后，查询请求状态：

```sql
SELECT id, status, attempts, last_error 
FROM certificate_requests 
WHERE id=2;
```

预期结果：
- status = failed（当attempts >= poll_max_attempts）
- attempts = 10（达到最大重试次数）
- last_error = "Failed to request certificate: ..."

### 验证进入 error 并可 retry

查询challenge TXT记录：

```sql
SELECT id, desired_state, owner_type, owner_id 
FROM domain_dns_records 
WHERE owner_type='acme_challenge' AND owner_id=2;
```

预期结果：
- desired_state = absent（自动清理）

执行retry：

```bash
curl -X POST "http://localhost:8080/api/v1/acme/certificate/retry" \
  -H "Content-Type: application/json" \
  -d '{"id": 2}'
```

查询retry后状态：

```sql
SELECT id, status, attempts, last_error 
FROM certificate_requests 
WHERE id=2;
```

预期结果：
- status = pending（retry成功）
- attempts = 10（保持不变）
- last_error = "..."（保留）

### 回滚步骤

1. Git回滚
```bash
cd /home/ubuntu/go_cmdb_new
git revert 522b45b
git push origin main
```

2. 停止Worker
```bash
# 修改配置
export ACME_WORKER_ENABLED=false
# 重启控制端
pkill -f "bin/cmdb"
./bin/cmdb &
```

3. 清理challenge记录（可选）
```sql
DELETE FROM domain_dns_records 
WHERE owner_type='acme_challenge';
```

4. 影响说明
回滚后系统退回到"证书success不自动apply"的旧行为：
- 证书签发成功后需要人工调用/api/v1/config/apply
- retry会清空attempts和last_error（审计失真）
- failed请求不会自动清理challenge TXT（可能污染后续签发）

---

## 9. 已知限制

1. 未实现自动续期
   - 当前仅支持手动请求证书
   - 自动续期功能将在T2-06实现

2. 全量下发策略
   - OnCertificateIssued触发的apply_config是全量下发（所有enabled nodes）
   - 未实现按website或line_group的精细化下发

3. 幂等性依赖reason字段
   - 如果手动修改config_versions.reason，可能导致重复下发
   - 建议不要手动修改reason字段

4. Challenge清理依赖DNS Worker
   - 如果DNS Worker未启用，challenge TXT记录不会被删除
   - 建议确保DNS Worker正常运行

5. 错误信息截断
   - last_error字段限制500字符
   - 超长错误信息会被截断

---

## 10. 统计数据

| 指标 | 数量 |
|-----|------|
| 修改文件 | 2 |
| 新增文件 | 1 |
| 新增方法 | 2 |
| 修改方法 | 2 |
| 新增代码行 | ~150 |
| curl测试 | 20 |
| SQL验证 | 15 |

---

## 11. 验收清单

- [x] P0-01: 证书签发成功后自动apply_config
- [x] P0-02: retry API语义修正（不清attempts）
- [x] P0-03: failed超限自动清理challenge TXT
- [x] config_versions自动生成（reason = acme-issued:{certID}）
- [x] agent_tasks自动生成（type = apply_config）
- [x] website_https.certificate_id自动更新
- [x] certificate_bindings自动激活
- [x] retry保留attempts和last_error
- [x] failed超限后challenge TXT自动清理
- [x] 幂等性保证（同reason不重复）
- [x] 验收测试脚本（20 curl + 15 SQL）
- [x] 代码编译通过
- [x] 代码提交GitHub（522b45b）
- [x] 完整交付报告

---

## 12. 交付完成

T2-05-fix任务已完整交付，所有P0改造均已完成，包括：

1. P0-01: 证书签发成功后自动apply_config
2. P0-02: retry API语义修正（不清attempts）
3. P0-03: failed超限自动清理challenge TXT
4. 验收测试: 20条curl + 15条SQL
5. 交付报告: 完整文档（无图标/emoji）

所有要求均已满足，可以进行验收。

---

交付人: Manus AI
交付日期: 2026-01-23
GitHub: https://github.com/labubu-daydayone/go_cmdb/commit/522b45b
