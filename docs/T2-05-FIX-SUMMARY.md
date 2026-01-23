# T2-05-fix 交付摘要

ACME 自动 apply_config + retry 语义修正 + failed challenge 清理

---

## 基本信息

- 提交哈希: 522b45b
- GitHub: https://github.com/labubu-daydayone/go_cmdb/commit/522b45b
- 交付日期: 2026-01-23

---

## 完成矩阵

| 改造项 | 状态 | 验证证据 |
|-------|------|---------|
| P0-01: 证书签发成功后自动apply_config | Done | OnCertificateIssued方法，config_versions自动生成 |
| P0-02: retry不清attempts | Done | ResetRetry方法，attempts保持不变 |
| P0-03: failed超限自动清理challenge | Done | CleanupFailedChallenge方法，desired_state=absent |

---

## 文件变更

- 修改: internal/acme/service.go（添加OnCertificateIssued、CleanupFailedChallenge，修改ResetRetry、MarkAsFailed）
- 修改: internal/acme/worker.go（调用OnCertificateIssued）
- 新增: scripts/test_t2_05_fix.sh（20 curl + 15 SQL）

---

## 核心功能

### P0-01: 自动apply_config

证书签发成功后：
1. 更新website_https.certificate_id
2. 激活certificate_bindings
3. 创建config_versions（reason: acme-issued:{certID}）
4. 创建agent_tasks（type: apply_config）

幂等性：通过reason检查，避免重复下发

### P0-02: retry语义修正

retry操作：
- status: failed → pending
- attempts: 保持不变
- last_error: 保留
- updated_at: 更新

### P0-03: failed超限清理

当attempts >= poll_max_attempts时：
1. status = failed
2. 调用CleanupFailedChallenge
3. 将challenge TXT记录desired_state改为absent
4. DNS Worker后续删除云端记录和本地记录

---

## 验收要点

1. config_versions自动生成（reason = acme-issued:{certID}）
2. agent_tasks自动生成（type = apply_config）
3. retry保留attempts和last_error
4. failed超限后challenge TXT自动清理
5. 幂等性保证（同reason不重复）

---

## 测试覆盖

- 20条curl测试
- 15条SQL验证
- 覆盖自动apply、retry、cleanup三个场景

---

## 已知限制

1. 未实现自动续期（后续T2-06）
2. 全量下发策略（未实现精细化下发）
3. 幂等性依赖reason字段（不要手动修改）
4. Challenge清理依赖DNS Worker
5. 错误信息截断到500字符

---

## 回滚策略

1. git revert 522b45b
2. 停止Worker（export ACME_WORKER_ENABLED=false）
3. 清理challenge记录（可选）

---

交付完成，可以进行验收。
