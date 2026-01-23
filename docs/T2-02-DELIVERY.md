# T2-02 mTLS双向认证交付报告

## 1. 提交信息

**任务名称**: T2-02 控制端↔Agent mTLS双向认证  
**交付日期**: 2026-01-23  
**状态**: ✅ 完成

## 2. 文件变更清单

### 新增文件

| 文件路径 | 说明 |
|---------|------|
| `api/v1/agent_identities/handler.go` | Agent Identity管理API（create/revoke/list） |
| `scripts/test_mtls.sh` | mTLS验收测试脚本（12个测试场景 + 5条SQL验证） |
| `docs/T2-02-DELIVERY.md` | 本交付报告 |

### 修改文件

| 文件路径 | 变更说明 |
|---------|---------|
| `cmd/agent/main.go` | 改造为HTTPS服务器，强制mTLS（tls.RequireAndVerifyClientCert） |
| `agent/api/v1/router.go` | 移除Bearer Token验证逻辑 |
| `internal/agent/client.go` | 已支持mTLS客户端证书（无需修改） |
| `internal/agent/dispatcher.go` | 已添加agent_identity验证（无需修改） |
| `api/v1/router.go` | 添加agent-identities路由（需admin权限） |
| `api/v1/middleware/auth.go` | 添加AdminRequired中间件 |
| `api/v1/agent_tasks/handler.go` | 修改NewHandler签名，传递*config.Config而非agentToken |

### 已存在文件（无需修改）

| 文件路径 | 说明 |
|---------|------|
| `internal/model/agent_identity.go` | Agent身份模型（已完成） |
| `scripts/generate_certs.sh` | 证书生成脚本（已完成） |
| `internal/config/config.go` | mTLS配置项（已完成） |

## 3. 路由变更清单

### 新增路由（需admin权限）

| 方法 | 路径 | 说明 | 权限 |
|-----|------|------|------|
| GET | `/api/v1/agent-identities` | 查询agent identities（支持筛选） | admin |
| POST | `/api/v1/agent-identities/create` | 创建agent identity | admin |
| POST | `/api/v1/agent-identities/revoke` | 吊销agent identity | admin |

### 修改路由

| 方法 | 路径 | 变更说明 |
|-----|------|---------|
| POST | `/agent/v1/tasks/execute` | 移除Bearer Token验证，改为mTLS认证 |

## 4. 证书生成与启动步骤

### 4.1 生成证书

```bash
cd /home/ubuntu/go_cmdb_new
bash scripts/generate_certs.sh
```

**输出示例**：
```
=== Certificate Generation Complete ===
CA Certificate:      ./certs/ca/ca-cert.pem
Control Client Cert: ./certs/control/client-cert.pem
Control Client Key:  ./certs/control/client-key.pem
Agent Server Cert:   ./certs/agent/server-cert.pem
Agent Server Key:    ./certs/agent/server-key.pem
Control Fingerprint: C6:FA:77:7A:82:67:ED:EF:6B:5E:DF:58:6C:0A:6A:4D:56:85:1D:70:E9:62:48:7F:52:2B:81:CA:B2:39:3A:7F
Agent Fingerprint:   A0:13:FD:0D:04:A6:46:1E:10:AD:43:87:6D:63:69:28:7B:E6:8F:A3:4E:A8:24:BA:C4:BE:B2:54:04:3C:9B:E6
```

**记录Agent Fingerprint**，后续创建agent_identity时需要。

### 4.2 启动Agent

```bash
cd /home/ubuntu/go_cmdb_new

# 设置环境变量
export AGENT_HTTP_ADDR=":9090"
export AGENT_CERT="./certs/agent/server-cert.pem"
export AGENT_KEY="./certs/agent/server-key.pem"
export AGENT_CA="./certs/ca/ca-cert.pem"

# 启动Agent
./bin/agent
```

**预期输出**：
```
Starting agent server with mTLS...
HTTPS address: :9090
Agent certificate fingerprint (SHA256): a013fd0d04a6461e10ad43876d6369287be68fa34ea824bac4beb254043c9be6
Agent server running on :9090 (HTTPS with mTLS)
```

### 4.3 启动控制端

```bash
cd /home/ubuntu/go_cmdb_new

# 设置环境变量（.env文件）
# 确保以下配置存在：
# MTLS_ENABLED=true
# CA_CERT_PATH=./certs/ca/ca-cert.pem
# CLIENT_CERT_PATH=./certs/control/client-cert.pem
# CLIENT_KEY_PATH=./certs/control/client-key.pem

# 启动控制端
./bin/cmdb
```

**预期输出**：
```
Starting CMDB server...
Database connected successfully
Server running on :8080
```

### 4.4 创建Agent Identity

```bash
# 1. 登录获取token
TOKEN=$(curl -s -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

# 2. 创建节点
NODE_ID=$(curl -s -X POST "http://localhost:8080/api/v1/nodes/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-node-1",
    "mainIp": "127.0.0.1",
    "agentPort": 9090,
    "location": "local",
    "isp": "test",
    "status": "active"
  }' | jq -r '.data.id')

# 3. 创建agent identity（使用Agent Fingerprint）
curl -X POST "http://localhost:8080/api/v1/agent-identities/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"nodeId\": ${NODE_ID},
    \"certFingerprint\": \"A0:13:FD:0D:04:A6:46:1E:10:AD:43:87:6D:63:69:28:7B:E6:8F:A3:4E:A8:24:BA:C4:BE:B2:54:04:3C:9B:E6\"
  }"
```

## 5. curl命令集合（10+条）

### 5.1 认证相关

```bash
# 1. 登录获取token
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 5.2 Agent Identity管理

```bash
# 2. 查询所有identities
curl -X GET "http://localhost:8080/api/v1/agent-identities" \
  -H "Authorization: Bearer ${TOKEN}"

# 3. 按nodeId筛选
curl -X GET "http://localhost:8080/api/v1/agent-identities?nodeId=1" \
  -H "Authorization: Bearer ${TOKEN}"

# 4. 按status筛选
curl -X GET "http://localhost:8080/api/v1/agent-identities?status=active" \
  -H "Authorization: Bearer ${TOKEN}"

# 5. 创建identity（成功）
curl -X POST "http://localhost:8080/api/v1/agent-identities/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "certFingerprint": "A0:13:FD:0D:04:A6:46:1E:10:AD:43:87:6D:63:69:28:7B:E6:8F:A3:4E:A8:24:BA:C4:BE:B2:54:04:3C:9B:E6"
  }'

# 6. 重复创建identity（失败 - 已存在）
curl -X POST "http://localhost:8080/api/v1/agent-identities/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "certFingerprint": "A0:13:FD:0D:04:A6:46:1E:10:AD:43:87:6D:63:69:28:7B:E6:8F:A3:4E:A8:24:BA:C4:BE:B2:54:04:3C:9B:E6"
  }'

# 7. 创建identity（失败 - 节点不存在）
curl -X POST "http://localhost:8080/api/v1/agent-identities/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 99999,
    "certFingerprint": "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99"
  }'

# 8. 吊销identity
curl -X POST "http://localhost:8080/api/v1/agent-identities/revoke" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1}'

# 9. 重复吊销identity（失败 - 已吊销）
curl -X POST "http://localhost:8080/api/v1/agent-identities/revoke" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1}'
```

### 5.3 任务下发测试

```bash
# 10. 创建任务（identity active - 成功）
curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "apply_config",
    "payload": {"test": "data"}
  }'

# 11. 创建任务（identity revoked - 失败）
# 先revoke identity，再创建任务
curl -X POST "http://localhost:8080/api/v1/agent-identities/revoke" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"nodeId": 1}'

curl -X POST "http://localhost:8080/api/v1/agent-tasks/create" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "nodeId": 1,
    "type": "reload",
    "payload": {}
  }'

# 12. 查询任务状态
curl -X GET "http://localhost:8080/api/v1/agent-tasks/1" \
  -H "Authorization: Bearer ${TOKEN}"
```

### 5.4 直接访问Agent（mTLS测试）

```bash
# 13. 使用控制端证书直接访问Agent（成功）
curl -k \
  --cert ./certs/control/client-cert.pem \
  --key ./certs/control/client-key.pem \
  --cacert ./certs/ca/ca-cert.pem \
  -X POST "https://localhost:9090/agent/v1/tasks/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "test-direct-123",
    "type": "apply_config",
    "payload": {"direct": true}
  }'

# 14. 不使用证书访问Agent（失败 - mTLS握手失败）
curl -k \
  -X POST "https://localhost:9090/agent/v1/tasks/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "test-no-cert",
    "type": "apply_config",
    "payload": {}
  }'
```

## 6. SQL验证

### 6.1 查询agent_identities表

```sql
-- 查询所有identities
SELECT id, node_id, cert_fingerprint, status, issued_at, revoked_at 
FROM agent_identities;

-- 查询active identities
SELECT id, node_id, cert_fingerprint, status 
FROM agent_identities 
WHERE status = 'active';

-- 查询revoked identities
SELECT id, node_id, cert_fingerprint, status, revoked_at 
FROM agent_identities 
WHERE status = 'revoked';

-- 按节点查询
SELECT id, node_id, cert_fingerprint, status 
FROM agent_identities 
WHERE node_id = 1;

-- 统计各状态数量
SELECT status, COUNT(*) as count 
FROM agent_identities 
GROUP BY status;
```

### 6.2 查询agent_tasks表

```sql
-- 查询所有任务
SELECT id, node_id, type, status, last_error, attempts, created_at 
FROM agent_tasks 
ORDER BY id DESC 
LIMIT 10;

-- 查询成功任务
SELECT id, node_id, type, status, attempts 
FROM agent_tasks 
WHERE status = 'success';

-- 查询失败任务（identity验证失败）
SELECT id, node_id, type, status, last_error, attempts 
FROM agent_tasks 
WHERE status = 'failed' 
  AND last_error LIKE '%identity%';

-- 统计各状态任务数量
SELECT status, COUNT(*) as count 
FROM agent_tasks 
GROUP BY status;
```

### 6.3 关联查询

```sql
-- 查询节点及其identity状态
SELECT 
  n.id as node_id,
  n.name as node_name,
  n.main_ip,
  ai.cert_fingerprint,
  ai.status as identity_status,
  ai.issued_at,
  ai.revoked_at
FROM nodes n
LEFT JOIN agent_identities ai ON n.id = ai.node_id
WHERE n.status = 'active';

-- 查询节点的任务执行情况
SELECT 
  n.id as node_id,
  n.name as node_name,
  COUNT(CASE WHEN at.status = 'success' THEN 1 END) as success_count,
  COUNT(CASE WHEN at.status = 'failed' THEN 1 END) as failed_count,
  COUNT(CASE WHEN at.status = 'pending' THEN 1 END) as pending_count
FROM nodes n
LEFT JOIN agent_tasks at ON n.id = at.node_id
GROUP BY n.id, n.name;
```

## 7. 验收场景

### 7.1 成功场景

| 场景 | 前置条件 | 操作 | 预期结果 |
|-----|---------|------|---------|
| 正常任务下发 | identity active + mTLS证书正确 | 创建任务 | 任务status=success |
| 创建identity | 节点存在 + fingerprint唯一 | POST /agent-identities/create | 返回identity对象 |
| 查询identities | 存在identities | GET /agent-identities | 返回列表 |
| 吊销identity | identity active | POST /agent-identities/revoke | status变为revoked |
| 直接访问Agent | 使用控制端证书 | curl https://agent | mTLS握手成功，返回200 |

### 7.2 失败场景

| 场景 | 前置条件 | 操作 | 预期结果 |
|-----|---------|------|---------|
| identity不存在 | 节点无identity | 创建任务 | 任务status=failed，last_error="identity not found" |
| identity revoked | identity已吊销 | 创建任务 | 任务status=failed，last_error="identity is revoked" |
| 证书不信任 | Agent使用不被信任的证书 | 创建任务 | mTLS握手失败，任务status=failed |
| 无客户端证书 | 控制端不提供证书 | 访问Agent | mTLS握手失败，连接被拒绝 |
| 重复创建identity | identity已存在 | POST /agent-identities/create | 返回错误码，message="already exists" |
| 重复吊销identity | identity已revoked | POST /agent-identities/revoke | 返回错误码，message="already revoked" |
| 非admin访问 | 普通用户token | POST /agent-identities/create | 返回403 Forbidden |

## 8. 回滚策略

### 8.1 代码回滚

```bash
# 回滚到T2-01（Bearer Token版本）
cd /home/ubuntu/go_cmdb_new
git log --oneline | head -5  # 查看提交历史
git revert <T2-02-commit-hash>  # 回滚T2-02提交
```

### 8.2 配置回滚

```bash
# 1. 停止服务
pkill -f "bin/cmdb"
pkill -f "bin/agent"

# 2. 修改.env，禁用mTLS
echo "MTLS_ENABLED=false" >> .env

# 3. 恢复Bearer Token配置
echo "AGENT_TOKEN=your-secret-token" >> .env

# 4. 重启服务（使用旧版本二进制）
./bin/cmdb.backup &
./bin/agent.backup &
```

### 8.3 数据库回滚

```sql
-- 删除agent_identities表数据（保留表结构）
TRUNCATE TABLE agent_identities;

-- 或删除表（如需完全回滚）
DROP TABLE IF EXISTS agent_identities;

-- 清理失败任务（可选）
DELETE FROM agent_tasks WHERE status = 'failed' AND last_error LIKE '%identity%';
```

### 8.4 证书清理

```bash
# 删除生成的证书
rm -rf /home/ubuntu/go_cmdb_new/certs
```

## 9. 已知限制

### 9.1 证书管理

- **证书过期处理**: 当前证书有效期为365天，过期后需要重新生成并更新agent_identities表
- **证书吊销列表（CRL）**: 未实现CRL机制，仅通过agent_identities表的status字段控制
- **证书轮换**: 不支持在线证书轮换，需要停机更新

### 9.2 性能限制

- **指纹计算**: 每次Agent启动都会计算证书指纹（SHA256），对性能影响可忽略
- **数据库查询**: dispatcher每次下发任务都会查询agent_identities表，高并发场景建议添加缓存

### 9.3 安全限制

- **CA证书保护**: CA私钥未加密存储，生产环境应使用HSM或密钥管理服务
- **证书指纹算法**: 当前使用SHA256，如需更强安全性可升级为SHA384/SHA512
- **mTLS超时**: 当前固定10秒超时，网络不稳定场景可能需要调整

### 9.4 功能限制

- **批量操作**: 不支持批量创建/吊销identities
- **审计日志**: 未记录identity创建/吊销操作的审计日志
- **通知机制**: identity被吊销时不会主动通知Agent

## 10. 后续优化建议

### 10.1 P1优先级（必须做）

- [x] 证书指纹计算统一（SHA256(DER)）
- [x] 本地联调验证（至少5个场景）
- [x] 编写10+条curl命令和SQL验证脚本

### 10.2 P2优先级（建议做）

- [ ] 添加identity操作审计日志（记录创建/吊销操作）
- [ ] 实现证书过期检查（定期任务检查证书有效期）
- [ ] 添加identity缓存（Redis）减少数据库查询
- [ ] 支持批量创建/吊销identities

### 10.3 P3优先级（可选）

- [ ] 实现CRL机制（证书吊销列表）
- [ ] 支持在线证书轮换
- [ ] 添加Prometheus metrics（mTLS握手成功率、失败原因统计）
- [ ] 实现identity自动续期机制

## 11. 测试执行

### 11.1 自动化测试

```bash
# 运行完整测试脚本
cd /home/ubuntu/go_cmdb_new
bash scripts/test_mtls.sh
```

### 11.2 手动测试

参考第5节"curl命令集合"和第6节"SQL验证"逐条执行。

## 12. 交付清单

- [x] Agent服务器改造（HTTPS + 强制mTLS）
- [x] 控制端client改造（mTLS客户端证书）
- [x] dispatcher改造（agent_identity验证）
- [x] Agent Identity管理API（create/revoke/list）
- [x] AdminRequired中间件
- [x] 证书生成脚本
- [x] 验收测试脚本（12个场景 + 5条SQL）
- [x] 交付报告（本文档）
- [x] 删除/停用AGENT_TOKEN逻辑
- [x] 证书指纹算法统一（SHA256）
- [x] 回滚策略
- [x] 已知限制说明

## 13. 签收确认

**开发者**: Manus AI  
**交付日期**: 2026-01-23  
**验收人**: _____________  
**验收日期**: _____________  
**验收结果**: □ 通过  □ 不通过  
**备注**: _____________

---

**附录A: 证书指纹格式说明**

- **格式**: SHA256(DER)，十六进制，冒号分隔
- **示例**: `A0:13:FD:0D:04:A6:46:1E:10:AD:43:87:6D:63:69:28:7B:E6:8F:A3:4E:A8:24:BA:C4:BE:B2:54:04:3C:9B:E6`
- **计算方法**: 
  ```go
  fingerprint := sha256.Sum256(cert.Certificate[0])
  fingerprintHex := hex.EncodeToString(fingerprint[:])
  ```

**附录B: 环境变量清单**

| 变量名 | 说明 | 示例值 |
|-------|------|--------|
| `MTLS_ENABLED` | 是否启用mTLS | `true` |
| `CA_CERT_PATH` | CA证书路径 | `./certs/ca/ca-cert.pem` |
| `CLIENT_CERT_PATH` | 控制端客户端证书路径 | `./certs/control/client-cert.pem` |
| `CLIENT_KEY_PATH` | 控制端客户端密钥路径 | `./certs/control/client-key.pem` |
| `AGENT_CERT` | Agent服务端证书路径 | `./certs/agent/server-cert.pem` |
| `AGENT_KEY` | Agent服务端密钥路径 | `./certs/agent/server-key.pem` |
| `AGENT_CA` | Agent CA证书路径 | `./certs/ca/ca-cert.pem` |
| `AGENT_HTTP_ADDR` | Agent监听地址 | `:9090` |
