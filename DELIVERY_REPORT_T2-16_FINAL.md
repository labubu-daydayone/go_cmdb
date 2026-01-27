# T2-16 ACME账号列表与管理API - 任务完成

## 任务性质

- 类型：功能补齐（P0）
- 模块：ACME账号管理
- 前置依赖：T2-16-fix-01已完成，T0-STD-01统一响应规范已生效

## 任务总结

成功实现ACME账号管理的5个核心接口，所有接口符合T0-STD-01统一响应规范，实现完整的业务约束保护。

## 实现的接口（5个）

### 1. 获取ACME账号列表
**接口**：GET /api/v1/acme/accounts

**查询参数**：
- provider_id (可选): 按CA过滤
- status (可选): active/disabled
- page (可选): 默认1
- page_size (可选): 默认20

**响应格式**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "provider_id": 1,
        "provider_name": "letsencrypt",
        "email": "admin@example.com",
        "status": "active",
        "is_default": true,
        "created_at": "2026-01-25T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

**实现要点**：
- ✅ 支持分页
- ✅ 支持provider_id和status过滤
- ✅ is_default通过acme_provider_defaults判断
- ✅ 返回data.items数组

### 2. 设置默认ACME账号
**接口**：POST /api/v1/acme/accounts/set-default

**请求参数**：
```json
{
  "providerId": 1,
  "accountId": 12
}
```

**业务规则**：
- ✅ 一个provider只能有一个default
- ✅ 使用acme_provider_defaults表
- ✅ 原子更新（Upsert）
- ✅ 被设置的账号必须是active

**响应格式**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "providerId": 1,
        "accountId": 12
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

### 3. 禁用ACME账号
**接口**：POST /api/v1/acme/accounts/disable

**请求参数**：
```json
{
  "id": 12
}
```

**业务规则**：
- ✅ 若该账号是default：禁止禁用，返回业务错误
- ✅ 仅更新status，不删除数据
- ✅ 幂等操作

**响应格式**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "status": "disabled"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**错误场景**：
```json
{
  "code": 3002,
  "message": "cannot disable default account, please set another account as default first",
  "data": null
}
```

### 4. 启用ACME账号
**接口**：POST /api/v1/acme/accounts/enable

**请求参数**：
```json
{
  "id": 12
}
```

**业务规则**：
- ✅ 仅允许从disabled → active
- ✅ 不自动设为default
- ✅ 幂等操作

**响应格式**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "status": "active"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

### 5. 删除ACME账号（受限）
**接口**：POST /api/v1/acme/accounts/delete

**请求参数**：
```json
{
  "id": 12
}
```

**删除约束**（必须校验）：
- ✅ 禁止删除default账号
- ✅ 禁止删除被certificate_requests引用的账号
- ✅ 禁止删除被certificates引用的账号

**响应格式**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "deleted": true
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 1
  }
}
```

**错误场景**：
```json
// Default账号
{
  "code": 3002,
  "message": "cannot delete default account, please set another account as default first",
  "data": null
}

// 被引用
{
  "code": 3002,
  "message": "cannot delete account, it is referenced by certificate requests",
  "data": null
}
```

## 实现约束

### 1. 统一响应规范
- ✅ 所有成功返回：data.items数组
- ✅ 所有失败返回：httpx.FailErr
- ✅ 禁止单对象返回
- ✅ 禁止自定义JSON

### 2. 查询实现要求
- ✅ 列表接口支持分页
- ✅ provider_name通过join获取
- ✅ is_default通过acme_provider_defaults判断

### 3. 文件范围
- api/v1/acme/handler.go - 接口实现
- api/v1/router.go - 路由配置
- 不新增表，不改表结构

## 验收标准（全部通过）

### 1. 编译
```bash
✅ go build ./... - 编译成功
```

### 2. 列表接口
```bash
curl http://localhost:8080/api/v1/acme/accounts

✅ 返回items数组
✅ is_default字段正确
✅ 支持分页和过滤
```

### 3. Default切换验证
```bash
✅ 切换后DB中acme_provider_defaults只有一条
✅ 列表接口实时反映变化
```

### 4. 禁用/删除校验
```bash
✅ default账号不可禁用
✅ default账号不可删除
✅ 被引用账号不可删除
```

## 代码修改

### 修改文件
- api/v1/acme/handler.go
  - 新增DeleteAccount接口（73行）
  - 修复DisableAccount添加default保护（12行）
  - 修复EnableAccount返回items格式（2行）
  - 修复SetDefault返回items格式（5行）

- api/v1/router.go
  - 添加DeleteAccount路由（1行）

### 代码统计
- 新增：约90行
- 修改：约20行

## 技术亮点

1. **完整的业务约束**：default账号保护、引用检查
2. **统一响应格式**：所有接口返回data.items
3. **幂等操作**：启用/禁用支持重复调用
4. **原子更新**：SetDefault使用Upsert语义
5. **安全删除**：多重检查防止误删

## 部署信息

- **代码已提交**：GitHub commit 9668817
- **服务状态**：编译通过
- **数据库**：使用已有表，无需迁移

## 回滚策略

如需回滚：
```bash
git revert 9668817
go build ./...
```

## 不在本次范围

以下内容全部不做：
- ❌ 新增ACME表
- ❌ 改动证书签发流程
- ❌ 写任何前端代码
- ❌ 文档补充

---

T2-16任务已完成，ACME账号管理API已全部实现并符合T0-STD-01规范，可投入生产使用。
