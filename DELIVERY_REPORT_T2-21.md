# T2-21 交付报告：证书生命周期列表统一唯一标识

## 任务概述

为证书生命周期列表接口（GET /api/v1/certificates）添加统一的字符串唯一标识字段，解决前端渲染时主键冲突问题。本次破例使用string类型id，仅限于此接口。

## 破例范围声明

**重要：本次修改仅限于证书生命周期列表接口，不影响系统其他部分**

- 仅在GET /api/v1/certificates接口中新增展示层字段
- 不修改、不替代、不影响任何业务主键（int类型）
- 其他所有列表接口仍必须使用int主键
- 此id字段仅用于前端渲染，不具备业务含义
- 禁止将该string id推广到其他业务列表
- 禁止将该string id作为任何写接口的入参

## 实施内容

### 1. 新增ID字段

在CertificateLifecycleItem结构体中新增ID字段（string类型）：

```go
type CertificateLifecycleItem struct {
    ID              string     `json:"id"` // Unique string identifier: "cert:<id>" or "req:<id>" (display-only, not a business key)
    ItemType        string     `json:"itemType"` // "certificate" | "request"
    CertificateID   *int       `json:"certificateId"`
    RequestID       *int       `json:"requestId"`
    // ... 其他字段
}
```

### 2. ID生成规则

根据itemType生成不同格式的ID：

- **certificate类型**：`id = "cert:" + strconv.Itoa(certificateId)`
- **request类型**：`id = "req:" + strconv.Itoa(requestId)`

示例：
- cert:2（证书ID=2）
- req:6（申请ID=6）

### 3. 字段互斥约束修正

修正certificate和request行的字段互斥关系：

**certificate行**：
- certificateId：必须有值
- requestId：必须为null

**request行**：
- requestId：必须有值
- certificateId：仅当status="issued"且有resultCertificateId时才有值，否则为null

代码实现：

```go
// Request item
item := CertificateLifecycleItem{
    ID:            "req:" + strconv.Itoa(req.ID),
    ItemType:      "request",
    RequestID:     &req.ID,
    Status:        mappedStatus,
    // ... 其他字段
}

// Field mutual exclusion: request rows should have certificateId only if issued
if mappedStatus == "issued" && req.ResultCertificateID != nil {
    item.CertificateID = req.ResultCertificateID
} else {
    item.CertificateID = nil
}

// Certificate item
items = append(items, CertificateLifecycleItem{
    ID:            "cert:" + strconv.Itoa(cert.ID),
    ItemType:      "certificate",
    CertificateID: &cert.ID,
    RequestID:     nil, // Field mutual exclusion: certificate rows must have requestId = null
    // ... 其他字段
})
```

## 验收测试

### 验收1：字段存在性与格式

```bash
curl -s "http://20.2.140.226:8080/api/v1/certificates?page=1&pageSize=20" \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ'
```

响应示例（前3条）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "req:6",
        "itemType": "request",
        "requestId": 6,
        "certificateId": 2,
        "status": "issued",
        "domains": ["4pxtech.com"],
        "fingerprint": "",
        "issueAt": null,
        "expireAt": null,
        "lastError": "",
        "createdAt": "2026-01-27T04:26:11.932+08:00",
        "updatedAt": "2026-01-27T04:31:16.807+08:00"
      },
      {
        "id": "req:5",
        "itemType": "request",
        "requestId": 5,
        "certificateId": null,
        "status": "failed",
        "domains": ["4pxtech.com"],
        "fingerprint": "",
        "issueAt": null,
        "expireAt": null,
        "lastError": "Failed to request certificate: ...",
        "createdAt": "2026-01-27T04:18:35.143+08:00",
        "updatedAt": "2026-01-27T04:25:09.794+08:00"
      },
      {
        "id": "req:4",
        "itemType": "request",
        "requestId": 4,
        "certificateId": null,
        "status": "failed",
        "domains": ["4pxtech.com"],
        "fingerprint": "",
        "issueAt": null,
        "expireAt": null,
        "lastError": "Failed to request certificate: ...",
        "createdAt": "2026-01-27T04:18:20.077+08:00",
        "updatedAt": "2026-01-27T04:24:29.87+08:00"
      }
    ],
    "total": 7,
    "page": 1,
    "pageSize": 20
  }
}
```

通过标准：
- code = 0 ✓
- data.items为数组 ✓
- 每一项都存在id字段 ✓
- id格式符合要求（cert:<int>或req:<int>）✓

### 验收2：唯一性与字段互斥

完整列表查询：

```bash
curl -s "http://20.2.140.226:8080/api/v1/certificates?page=1&pageSize=20" \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoiZ29fY21kYiIsImV4cCI6MTc2OTU0NTA4NSwiaWF0IjoxNzY5NDU4Njg1fQ.4jSrHxRlXqVbkhTpp0BDkUz7f4sFP7BsBbmvbiYDOqQ' | jq '.data.items | map({id, itemType, requestId, certificateId})'
```

响应：

```json
[
  {
    "id": "req:6",
    "itemType": "request",
    "requestId": 6,
    "certificateId": 2
  },
  {
    "id": "req:5",
    "itemType": "request",
    "requestId": 5,
    "certificateId": null
  },
  {
    "id": "req:4",
    "itemType": "request",
    "requestId": 4,
    "certificateId": null
  },
  {
    "id": "req:3",
    "itemType": "request",
    "requestId": 3,
    "certificateId": null
  },
  {
    "id": "req:2",
    "itemType": "request",
    "requestId": 2,
    "certificateId": null
  },
  {
    "id": "req:1",
    "itemType": "request",
    "requestId": 1,
    "certificateId": null
  },
  {
    "id": "cert:2",
    "itemType": "certificate",
    "requestId": null,
    "certificateId": 2
  }
]
```

通过标准：
- 同一certificateId和requestId不产生冲突（cert:2和req:2是不同记录）✓
- certificate行：requestId=null，certificateId有值 ✓
- request行（未签发）：requestId有值，certificateId=null ✓
- request行（已签发）：requestId有值，certificateId有值 ✓
- id稳定性：多次请求同一记录id不变 ✓

## 代码修改清单

修改文件：
- api/v1/cert/handler_list_lifecycle.go

修改内容：
1. CertificateLifecycleItem结构体新增ID字段（第17行）
2. Request item生成逻辑添加ID赋值（第114行）
3. Request item字段互斥约束修正（第125-130行）
4. Certificate item生成逻辑添加ID赋值（第148行）
5. Certificate item requestId强制为nil（第151行）

## 代码提交

- 提交哈希：71e233ad6307e2bfa27e13dd9b923703c5f0dacc
- 提交信息：T2-21: Add string id field to certificate lifecycle list (exception for this API only)
- 仓库：https://github.com/labubu-daydayone/go_cmdb
- 分支：main

## 部署状态

测试环境：20.2.140.226:8080

- 服务状态：运行中
- 启动命令：`./cmdb --config=config.ini`
- 部署时间：2026-01-27 05:11

## 测试通过情况

- 字段存在性测试：通过 ✓
- ID格式验证：通过 ✓
- 唯一性验证：通过 ✓
- 字段互斥约束：通过 ✓
- 稳定性验证：通过 ✓

## 回滚策略

本次修改仅涉及接口返回字段，无数据库变更：

```bash
git revert 71e233ad6307e2bfa27e13dd9b923703c5f0dacc
```

不涉及数据库迁移回滚。

## 防扩散说明

**系统业务主键仍全部使用int类型**

本次破例仅限于证书生命周期列表接口（GET /api/v1/certificates），原因如下：

1. 该接口同时返回两类数据（certificate_requests和certificates）
2. 两类数据主键空间不同但数值可能冲突（如requestId=2和certificateId=2）
3. 前端需要稳定的全局唯一行标识用于渲染、diff、操作
4. 此id仅用于前端展示，不具备业务含义，不参与任何写操作

**其他所有接口必须继续使用int主键，禁止推广此模式。**

## 遗留问题

无

## 备注

1. 启动服务时必须使用`--config=config.ini`参数指定配置文件
2. ID字段仅用于前端渲染，后端写操作仍使用certificateId和requestId
3. 此破例模式不得推广到其他业务列表接口
