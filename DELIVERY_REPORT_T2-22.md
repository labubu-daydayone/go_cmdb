# T2-22 交付报告：删除失败的证书申请（P0生产止血）

## 任务概述

实现删除失败证书申请的接口，用于清理无业务价值的failed request记录，终止不可恢复的ACME申请流程。

## 任务级别

P0（生产止血）

## 实施内容

### 1. 接口实现

创建了DELETE接口handler：`api/v1/cert/handler_delete_failed_request.go`

接口路径：`POST /api/v1/acme/certificate/requests/:requestId/delete`

### 2. 删除条件（强约束）

只允许删除同时满足以下所有条件的记录：

1. itemType = request
2. status = failed
3. certificateId IS NULL（未生成任何证书）

### 3. 校验逻辑

按顺序执行以下校验：

```go
// 1. 查询certificate_requests表
var request model.CertificateRequest
if err := h.db.Where("id = ?", requestID).First(&request).Error; err != nil {
    // 返回404：certificate request not found
}

// 2. 验证status必须为"failed"
if request.Status != "failed" {
    // 返回3003：only failed certificate request can be deleted
}

// 3. 验证result_certificate_id必须为NULL
if request.ResultCertificateID != nil {
    // 返回3003：certificate already issued, request cannot be deleted
}

// 4. 物理删除
h.db.Delete(&request)
```

### 4. 删除行为

- 物理删除certificate_requests记录
- 不保留归档、不软删除
- 删除即表示"终止该申请生命周期"

### 5. ACME Worker行为验证

确认ACME Worker查询条件：

```go
// internal/acme/service.go:GetPendingRequests
Where("status IN (?, ?)", model.CertificateRequestStatusPending, model.CertificateRequestStatusRunning)
```

结论：Worker只处理status为"pending"或"running"的记录，被删除的failed记录不会再次被处理。

## 验收测试

### 正向验收1：删除failed request（requestId=4）

```bash
curl -X POST "http://20.2.140.226:8080/api/v1/acme/certificate/requests/4/delete" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ"
```

响应：

```json
{
  "code": 0,
  "message": "certificate request terminated",
  "data": {
    "requestId": 4
  }
}
```

通过标准：
- code = 0 ✓
- 证书列表中不再出现requestId=4 ✓

### 正向验收2：删除另一个failed request（requestId=5）

```bash
curl -X POST "http://20.2.140.226:8080/api/v1/acme/certificate/requests/5/delete" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ"
```

响应：

```json
{
  "code": 0,
  "message": "certificate request terminated",
  "data": {
    "requestId": 5
  }
}
```

通过标准：
- code = 0 ✓
- 证书列表中不再出现requestId=5 ✓

### 验证列表状态

删除前：7条记录（6个request + 1个certificate）
删除后：5条记录（4个request + 1个certificate）

```bash
curl -s "http://20.2.140.226:8080/api/v1/certificates?page=1&pageSize=20" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ"
```

响应：

```json
{
  "total": 5,
  "items": [
    {
      "id": "req:6",
      "requestId": 6,
      "certificateId": 2,
      "status": "issued"
    },
    {
      "id": "req:3",
      "requestId": 3,
      "certificateId": null,
      "status": "failed"
    },
    {
      "id": "req:2",
      "requestId": 2,
      "certificateId": null,
      "status": "failed"
    },
    {
      "id": "req:1",
      "requestId": 1,
      "certificateId": null,
      "status": "failed"
    },
    {
      "id": "cert:2",
      "requestId": null,
      "certificateId": 2,
      "status": "valid"
    }
  ]
}
```

通过标准：
- requestId=4和5已从列表中移除 ✓
- 其他记录不受影响 ✓
- total从7减少到5 ✓

### 反向验收1：尝试删除issued request（应该失败）

```bash
curl -X POST "http://20.2.140.226:8080/api/v1/acme/certificate/requests/6/delete" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ"
```

响应：

```json
{
  "code": 3003,
  "message": "only failed certificate request can be deleted",
  "data": null
}
```

通过标准：
- 返回业务错误code=3003 ✓
- 错误信息明确 ✓
- 记录未被删除 ✓

### 反向验收2：尝试删除不存在的request

```bash
curl -X POST "http://20.2.140.226:8080/api/v1/acme/certificate/requests/999/delete" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ"
```

响应：

```json
{
  "code": 1004,
  "message": "certificate request not found",
  "data": null
}
```

通过标准：
- 返回404错误code=1004 ✓
- 错误信息明确 ✓

## 代码修改清单

新增文件：
- api/v1/cert/handler_delete_failed_request.go（87行）

修改文件：
- api/v1/router.go（添加路由注册）

## 代码提交

- 提交哈希：fd94be586e1650c67a3a592a6c9ac14dcd927d3f
- 提交信息：T2-22: Add delete failed certificate request API (P0 production fix)
- 仓库：https://github.com/labubu-daydayone/go_cmdb
- 分支：main

## 部署状态

测试环境：20.2.140.226:8080

- 服务状态：运行中
- 启动命令：`./cmdb --config=config.ini`
- 部署时间：2026-01-27 05:50

## 任务完成判定

- failed + certificateId=null 的request可被成功删除 ✓
- 删除后Worker不再处理 ✓（Worker只查询pending/running状态）
- 列表不再显示该记录 ✓
- 其它状态request / certificate不受影响 ✓

## 回滚策略

如需回滚：

```bash
git revert fd94be586e1650c67a3a592a6c9ac14dcd927d3f
```

重启服务即可。不需要数据恢复（删除的均为failed且无证书的记录）。

## 定死规则（防扩散）

**这是唯一允许删除request的场景**

- 后续不得扩展为"删除issuing / pending"
- 如需更多清理能力，必须新开任务卡
- 此接口仅用于清理垃圾数据，不是常规业务功能

## 前端配合要求

前端仅在以下条件下显示"删除"按钮：

```javascript
itemType === "request" && status === "failed" && certificateId === null
```

删除成功后：
- 该行从证书列表中移除
- 不需要刷新整页
- 不影响其他证书/申请显示

## 遗留问题

无

## 备注

1. 启动服务时必须使用`--config=config.ini`参数
2. 删除操作是物理删除，无法恢复
3. 仅删除failed且无证书的request，确保不会误删有价值数据
4. ACME Worker已验证不会处理failed状态的记录
