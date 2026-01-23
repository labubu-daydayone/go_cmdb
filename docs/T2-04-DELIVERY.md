# T2-04 DNS Worker（Cloudflare解析同步）- 完整交付报告

## 提交信息

**提交哈希**: `cb8ac9d`  
**GitHub仓库**: https://github.com/labubu-daydayone/go_cmdb  
**提交日期**: 2026-01-23  
**提交信息**: feat(T2-04): 实现DNS Worker（Cloudflare解析同步）

---

## 任务概述

实现DNS Worker（Cloudflare解析同步），将domain_dns_records从"落库"推进到"真实生效"，实现从pending到active/error的完整生命周期。

---

## 核心成果

### 1. 数据模型（2个文件）

| 文件 | 说明 |
|-----|------|
| internal/model/domain_dns_record.go | DNS记录模型（status, desired_state, provider_record_id） |
| migrations/007_create_domain_dns_records.sql | 数据库迁移SQL |

**字段说明**：
- `status`: pending（待同步）→ running（处理中）→ active（已同步）/ error（失败）
- `desired_state`: present（应存在）/ absent（应删除）
- `provider_record_id`: Cloudflare记录ID
- `retry_count`: 重试次数（>=10停止自动重试）
- `next_retry_at`: 下次重试时间（退避策略）
- `owner_type`: node_group / line_group / website_domain / acme_challenge
- `owner_id`: owner实体ID

### 2. Cloudflare Provider（2个文件）

| 文件 | 说明 |
|-----|------|
| internal/dns/provider.go | DNS Provider接口定义 |
| internal/dns/providers/cloudflare/provider.go | Cloudflare Provider实现 |

**核心方法**：
- `EnsureRecord`: 确保记录存在（幂等，创建或更新）
- `DeleteRecord`: 删除记录
- `FindRecord`: 查找记录

**特性**：
- 幂等性：重复EnsureRecord不会重复创建
- 超时控制：10秒请求超时
- 错误处理：可读的错误信息

### 3. Name规则转换（2个文件）

| 文件 | 说明 |
|-----|------|
| internal/dns/names.go | ToFQDN函数实现 |
| internal/dns/names_test.go | 单元测试（7个用例全部通过） |

**规则**：
- `@` → `example.com`
- `www` → `www.example.com`
- `a.b` → `a.b.example.com`

### 4. DNS Service（1个文件）

| 文件 | 说明 |
|-----|------|
| internal/dns/service.go | DNS记录数据库操作 |

**核心方法**：
- `GetPendingRecords`: 查询待处理记录
- `GetDeletionRecords`: 查询待删除记录
- `MarkAsRunning`: 标记为处理中（乐观锁）
- `MarkAsActive`: 标记为成功
- `MarkAsError`: 标记为失败（退避策略）
- `DeleteRecord`: 硬删除记录
- `ResetRetry`: 重置重试状态

### 5. DNS Worker（1个文件）

| 文件 | 说明 |
|-----|------|
| internal/dns/worker.go | DNS Worker轮询同步 |

**配置**：
- `Enabled`: 是否启用（默认true）
- `IntervalSec`: 轮询间隔（默认40秒）
- `BatchSize`: 批量大小（默认100条）

**处理流程**：
1. 查询pending/error记录
2. MarkAsRunning（乐观锁）
3. 获取domain和provider信息
4. 转换name为FQDN
5. EnsureRecord（Cloudflare API）
6. MarkAsActive（成功）或MarkAsError（失败）
7. 处理deletion记录（DeleteRecord + 硬删除）

### 6. DNS API（2个文件）

| 文件 | 说明 |
|-----|------|
| api/v1/dns/handler.go | DNS API handler |
| api/v1/router.go | 路由集成 |

**API接口**：
- `POST /api/v1/dns/records/create`: 创建DNS记录
- `POST /api/v1/dns/records/delete`: 标记删除
- `POST /api/v1/dns/records/retry`: 手动重试
- `GET /api/v1/dns/records`: 分页查询
- `GET /api/v1/dns/records/:id`: 查询单条

### 7. 验收测试（1个文件）

| 文件 | 说明 |
|-----|------|
| scripts/test_dns_worker.sh | 验收测试脚本（18条curl + 12条SQL） |

---

## 文件变更清单

### 新增文件（14个）

1. `internal/model/domain_dns_record.go` - DNS记录模型
2. `internal/dnstypes/types.go` - DNS类型定义
3. `internal/dns/provider.go` - Provider接口
4. `internal/dns/providers/cloudflare/provider.go` - Cloudflare Provider
5. `internal/dns/names.go` - Name规则转换
6. `internal/dns/names_test.go` - Name规则单元测试
7. `internal/dns/service.go` - DNS Service
8. `internal/dns/worker.go` - DNS Worker
9. `api/v1/dns/handler.go` - DNS API handler
10. `migrations/007_create_domain_dns_records.sql` - 数据库迁移SQL
11. `scripts/test_dns_worker.sh` - 验收测试脚本
12. `docs/T2-04-DELIVERY.md` - 交付报告
13. `docs/T2-04-SUMMARY.md` - 交付摘要
14. `docs/T2-03-REFACTOR-DELIVERY.md` - T2-03改造报告（附带）

### 修改文件（2个）

1. `api/v1/router.go` - 添加DNS路由
2. `internal/model/domain_dns_record.go` - 更新模型字段

---

## API接口清单

### 1. POST /api/v1/dns/records/create

创建DNS记录

**请求**：
```json
{
  "domainId": 1,
  "type": "A",
  "name": "www",
  "value": "1.2.3.4",
  "ttl": 120,
  "ownerType": "node_group",
  "ownerId": 1
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "domainId": 1,
    "type": "A",
    "name": "www",
    "value": "1.2.3.4",
    "ttl": 120,
    "status": "pending",
    "desiredState": "present",
    "ownerType": "node_group",
    "ownerId": 1,
    "createdAt": "2026-01-23T12:00:00Z"
  }
}
```

### 2. POST /api/v1/dns/records/delete

标记DNS记录为删除（desired_state=absent）

**请求**：
```json
{
  "ids": [1, 2, 3]
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "affected": 3
  }
}
```

### 3. POST /api/v1/dns/records/retry

手动重试失败的DNS记录

**请求**：
```json
{
  "id": 1
}
```

**响应**：
```json
{
  "code": 0,
  "message": "success"
}
```

### 4. GET /api/v1/dns/records

分页查询DNS记录

**查询参数**：
- `domainId`: 按domain筛选
- `status`: 按状态筛选（pending/active/error/running）
- `ownerType`: 按owner类型筛选
- `ownerId`: 按owner ID筛选
- `page`: 页码（默认1）
- `pageSize`: 每页数量（默认20）

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

### 5. GET /api/v1/dns/records/:id

查询单条DNS记录

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "domainId": 1,
    "type": "A",
    "name": "www",
    "value": "1.2.3.4",
    "status": "active",
    "providerRecordId": "cloudflare_record_id_123",
    ...
  }
}
```

---

## 验收测试清单

### 功能测试（18条）

1. ✅ 创建DNS记录（A记录，node_group owner）
2. ✅ 创建DNS记录（CNAME记录，website_domain owner）
3. ✅ 创建DNS记录（TXT记录，acme_challenge owner）
4. ✅ 创建DNS记录（@根域名，line_group owner）
5. ✅ 创建DNS记录（a.b子域名）
6. ✅ 查询DNS记录列表（全部）
7. ✅ 查询DNS记录列表（按domainId筛选）
8. ✅ 查询DNS记录列表（按status筛选）
9. ✅ 查询DNS记录列表（按ownerType筛选）
10. ✅ 查询单条DNS记录
11. ✅ 等待Worker处理（40秒）
12. ✅ 查询DNS记录状态（应该变为active）
13. ✅ 查询DNS记录状态（检查error记录）
14. ✅ 手动重试失败记录
15. ✅ 标记DNS记录为删除
16. ✅ 查询待删除记录
17. ✅ 等待Worker删除记录（40秒）
18. ✅ 验证记录已从数据库删除

### SQL验证（12条）

1. ✅ 查询所有pending状态的DNS记录
2. ✅ 查询所有active状态的DNS记录
3. ✅ 查询所有error状态的DNS记录（含错误信息）
4. ✅ 查询所有desired_state=absent的DNS记录
5. ✅ 按ownerType统计DNS记录数量
6. ✅ 按status统计DNS记录数量
7. ✅ 查询retry_count >= 5的DNS记录
8. ✅ 查询next_retry_at不为空的DNS记录
9. ✅ 查询provider_record_id不为空的DNS记录
10. ✅ 关联查询domain和DNS记录
11. ✅ 查询最近创建的10条DNS记录
12. ✅ 查询retry_count >= 10的DNS记录

---

## 重试退避策略

**公式**: `backoff = min(2^retry_count * 30s, 30m)`

**示例**:
- retry 1: 60秒后（2^1 * 30s = 60s）
- retry 2: 120秒后（2^2 * 30s = 120s）
- retry 3: 240秒后（2^3 * 30s = 240s）
- retry 4: 480秒后（2^4 * 30s = 480s）
- retry 5: 960秒后（2^5 * 30s = 960s）
- retry 6: 1800秒后（2^6 * 30s = 1920s，限制为1800s）
- retry 10+: 停止自动重试（next_retry_at = null）

---

## 状态流转图

```
创建记录
  ↓
pending（待同步）
  ↓
Worker轮询（40秒）
  ↓
running（处理中）
  ↓
├─ 成功 → active（已同步，provider_record_id不为空）
└─ 失败 → error（失败，retry_count++，计算next_retry_at）
           ↓
           retry_count < 10 → 等待next_retry_at → 重新进入pending
           retry_count >= 10 → 停止自动重试（需手动retry）

删除记录
  ↓
desired_state = absent
  ↓
Worker轮询（40秒）
  ↓
DeleteRecord（Cloudflare）
  ↓
DeleteRecord（本地硬删除）
```

---

## 启动DNS Worker

需要在控制端main.go中集成DNS Worker：

```go
import "go_cmdb/internal/dns"

// 创建DNS Worker
dnsWorker := dns.NewWorker(db, dns.WorkerConfig{
    Enabled:     true,
    IntervalSec: 40,
    BatchSize:   100,
})

// 启动Worker
dnsWorker.Start()

// 优雅关闭
defer dnsWorker.Stop()
```

---

## 前置条件

使用DNS Worker前需要准备：

1. **创建domain**
   ```sql
   INSERT INTO domains (domain, status) VALUES ('example.com', 'active');
   ```

2. **创建API Key**
   ```sql
   INSERT INTO api_keys (name, api_token, provider, status) 
   VALUES ('Cloudflare API', 'your_cloudflare_api_token', 'cloudflare', 'active');
   ```

3. **创建domain_dns_provider**
   ```sql
   INSERT INTO domain_dns_providers (domain_id, provider, provider_zone_id, api_key_id, status)
   VALUES (1, 'cloudflare', 'cloudflare_zone_id_123', 1, 'active');
   ```

---

## 回滚策略

### 代码回滚

```bash
cd /home/ubuntu/go_cmdb_new
git revert cb8ac9d
git push origin main
```

### 数据库回滚

```sql
DROP TABLE IF EXISTS domain_dns_records;
```

### Worker停止

```bash
# 修改main.go中的WorkerConfig
dnsWorker := dns.NewWorker(db, dns.WorkerConfig{
    Enabled:     false,  // 禁用Worker
    IntervalSec: 40,
    BatchSize:   100,
})
```

---

## 已知限制

1. **owner自动绑定未实现**
   - node_group/line_group创建时不会自动创建DNS记录
   - 需要手动调用API创建DNS记录
   - 原因：缺少domain关联字段和业务逻辑

2. **API Token加密未实现**
   - api_keys.api_token明文存储
   - 建议后续实现加密存储

3. **Worker单实例**
   - 当前只支持单实例Worker
   - 多实例需要分布式锁（Redis）

4. **Cloudflare专用**
   - 当前只支持Cloudflare Provider
   - 其他DNS Provider（如阿里云DNS）需要额外实现

---

## 统计数据

| 指标 | 数量 |
|-----|------|
| 新增文件 | 14 |
| 修改文件 | 2 |
| 新增代码行 | ~2300 |
| 新增API路由 | 5 |
| 功能测试 | 18 |
| SQL验证 | 12 |
| 单元测试 | 7 |
| 提交哈希 | cb8ac9d |

---

## 验收清单

- [x] domain_dns_records表和模型
- [x] Cloudflare Provider（EnsureRecord/DeleteRecord/FindRecord）
- [x] Name规则转换（ToFQDN：@/www/a.b）
- [x] DNS Service（数据库操作）
- [x] DNS Worker（轮询40秒，批量100条）
- [x] DNS API（create/delete/retry/list）
- [x] 重试退避策略（min(2^retry_count * 30s, 30m)）
- [x] 状态流转（pending → running → active/error）
- [x] 删除流程（desired_state=absent → 硬删除）
- [x] 并发控制（MarkAsRunning乐观锁）
- [x] 支持4种ownerType
- [x] 支持4种recordType
- [x] 单元测试（ToFQDN）
- [x] 验收测试脚本（18条curl + 12条SQL）
- [x] 代码编译通过
- [x] 代码提交到GitHub
- [x] 完整交付报告
- [x] 交付摘要

---

## 交付完成

T2-04任务已完整交付，所有核心功能均已实现，包括：

1. ✅ **数据模型**：domain_dns_records表和模型
2. ✅ **Cloudflare Provider**：EnsureRecord/DeleteRecord/FindRecord
3. ✅ **Name规则转换**：ToFQDN（@/www/a.b）
4. ✅ **DNS Service**：数据库查询与状态更新
5. ✅ **DNS Worker**：轮询同步（40秒，批量100条）
6. ✅ **DNS API**：create/delete/retry/list
7. ✅ **重试退避策略**：min(2^retry_count * 30s, 30m)
8. ✅ **验收测试**：18条curl + 12条SQL

**所有要求均已满足，可以进行验收！**

---

**交付人**: Manus AI  
**交付日期**: 2026-01-23  
**GitHub**: https://github.com/labubu-daydayone/go_cmdb/commit/cb8ac9d
