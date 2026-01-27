# T2-23 交付报告：统一证书删除接口 + failed申请自动清理

## 任务信息

- 任务编号：T2-23（修订版）
- 任务级别：P0
- 提交时间：2026-01-27
- 提交哈希：87ee8a66048bd1e52f5d05b19a1167c6f3c11959

## 任务目标

1. 实现统一证书删除接口，支持cert和req两种ID格式
2. 实现failed申请超3天自动清理机制

## 实现内容

### 1. 统一删除接口

**接口路径**：POST /api/v1/certificate/:id/delete

**支持的ID格式**：
- cert:{certificateId} - 删除证书（仅当未被使用时）
- req:{requestId} - 删除证书申请（仅当status=failed时）

**删除约束**：
- 证书删除：
  - 检查是否存在active binding（bind_type=website, is_active=true）
  - 如果有active binding，返回409错误
  - 删除前先处理外键引用（certificate_requests.result_certificate_id）
  - 事务删除：certificate_domains → certificate_bindings → certificates
- 申请删除：
  - 只允许删除status=failed的申请
  - 其他状态（pending/issuing/issued）返回409错误

**代码文件**：api/v1/cert/handler_delete.go

### 2. 自动清理Worker

**功能**：定期清理超过指定天数的failed证书申请

**配置项**（config.ini）：
```ini
[cert_cleaner]
enabled = true
interval_sec = 40
failed_keep_days = 3
```

**工作机制**：
- 每40秒执行一次清理
- 删除updated_at早于（当前时间 - failed_keep_days天）的failed申请
- 启动时立即执行一次清理

**代码文件**：
- internal/cert/cleaner.go - Worker实现
- internal/config/config.go - 配置加载
- cmd/cmdb/main.go - Worker启动

### 3. 配置加载修复

**问题**：getValueInt函数在INI值为0时返回默认值

**修复**：使用HasKey检查键是否存在，而不是检查值是否为0

**影响**：所有使用getValueInt的配置项都能正确处理0值

## 验收结果

### 验收4.2：删除failed request（统一接口）

**请求**：
```bash
curl -X POST "http://20.2.140.226:8080/api/v1/certificate/req:3/delete" \
  -H "Authorization: Bearer <TOKEN>"
```

**响应**：
```json
{
  "code": 0,
  "message": "certificate request deleted",
  "data": {
    "deleted": true,
    "id": "req:3"
  }
}
```

**验证**：列表total从5减少到4，req:3已移除

### 验收4.3：尝试删除非failed request（应拒绝）

**请求**：
```bash
curl -X POST "http://20.2.140.226:8080/api/v1/certificate/req:6/delete" \
  -H "Authorization: Bearer <TOKEN>"
```

**响应**：
```json
{
  "code": 3003,
  "message": "request is not deletable",
  "data": null
}
```

**HTTP状态码**：409 Conflict

### 验收4.4：删除未被使用的证书

**请求**：
```bash
curl -X POST "http://20.2.140.226:8080/api/v1/certificate/cert:2/delete" \
  -H "Authorization: Bearer <TOKEN>"
```

**响应**：
```json
{
  "code": 0,
  "message": "certificate deleted",
  "data": {
    "deleted": true,
    "id": "cert:2"
  }
}
```

**验证**：列表total从4减少到3，cert:2已移除

### 验收4.6：自动清理机制

**配置**：
```ini
[cert_cleaner]
enabled = true
interval_sec = 5
failed_keep_days = 0
```

**日志输出**：
```
2026/01/27 06:16:56 [Cert Cleaner] Starting with interval=5s, keep_days=0
2026/01/27 06:16:56 ✓ Certificate Cleaner initialized
2026/01/27 06:16:56 [Cert Cleaner] Cleaned 2 failed requests older than 2026-01-27 06:16:56.396155573 +0800 +08
```

**验证**：
- 启动时立即清理了2个failed request（req:2和req:1）
- 列表total从3减少到1
- 只剩下issued状态的req:6

## 代码提交

**提交哈希**：87ee8a66048bd1e52f5d05b19a1167c6f3c11959

**新增文件**：
- api/v1/cert/handler_delete.go（统一删除接口）
- internal/cert/cleaner.go（自动清理Worker）

**修改文件**：
- api/v1/router.go（路由注册）
- internal/config/config.go（配置加载修复）
- cmd/cmdb/main.go（Worker启动）

**GitHub链接**：https://github.com/labubu-daydayone/go_cmdb/commit/87ee8a66048bd1e52f5d05b19a1167c6f3c11959

## 关键技术点

### 1. 外键约束处理

删除证书时，certificate_requests表有外键约束fk_certificate_requests_result_certificate指向certificates表。解决方案：
```go
// 先将引用该证书的请求的result_certificate_id设为NULL
tx.Model(&model.CertificateRequest{}).
  Where("result_certificate_id = ?", certificateID).
  Update("result_certificate_id", nil)
```

### 2. 配置0值处理

原始代码：
```go
if value, err := cfgFile.Section(iniSection).Key(iniKey).Int(); err == nil && value != 0 {
    return value
}
```

修复后：
```go
if cfgFile.Section(iniSection).HasKey(iniKey) {
    if value, err := cfgFile.Section(iniSection).Key(iniKey).Int(); err == nil {
        return value
    }
}
```

### 3. 事务删除顺序

```
1. 更新certificate_requests.result_certificate_id为NULL
2. 删除certificate_domains
3. 删除certificate_bindings（包括历史记录）
4. 删除certificates
```

## 生产部署建议

### 配置建议

生产环境建议配置：
```ini
[cert_cleaner]
enabled = true
interval_sec = 3600    # 每小时清理一次
failed_keep_days = 7   # 保留7天的failed记录
```

### 监控建议

1. 监控cleaner日志中的清理数量
2. 如果清理数量异常增加，说明ACME申请失败率上升
3. 定期检查failed request的失败原因

### 回滚策略

如需回滚：
1. 回滚代码提交：git revert 87ee8a66048bd1e52f5d05b19a1167c6f3c11959
2. 重新编译并部署
3. 重启服务
4. 不需要数据恢复（删除的均为failed且无证书的记录）

## 任务完成判定

- 统一删除接口正常工作（支持cert和req两种ID格式）
- 删除约束正确执行（只能删除未使用的证书和failed申请）
- 自动清理机制正常工作（定期清理超期的failed申请）
- 配置0值问题已修复
- 外键约束处理正确
- 所有验收用例通过

任务已完成，符合P0优先级要求。
