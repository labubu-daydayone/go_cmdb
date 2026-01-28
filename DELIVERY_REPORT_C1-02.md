# 任务 C1-02 交付报告：Domain 下拉候选接口

## 任务信息

- 任务编号：C1-02
- 任务名称：Domain 下拉候选接口
- 任务级别：P0

## 任务目标

实现 Domain 下拉候选接口，用于前端选择器，只返回 status=active 且 purpose=cdn 的域名。

## 核心实现

### 1. DTO 结构体

创建文件：api/v1/domains/dto.go

```go
type DomainOptionDTO struct {
    ID        int64     `json:"id"`
    Domain    string    `json:"domain"`
    Status    string    `json:"status"`
    Purpose   string    `json:"purpose"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

所有字段使用 lowerCamelCase 命名。

### 2. Handler 实现

创建文件：api/v1/domains/handler_options.go

接口：GET /api/v1/domains/options

功能：
- 查询 status=active 且 purpose=cdn 的域名
- 按 id 降序排列
- 返回 data.items 结构

### 3. 路由注册

修改文件：api/v1/router.go

在 domains 路由组中添加：
```go
domainsOptionsHandler := domains.NewOptionsHandler(db)
domainsGroup.GET("/options", domainsOptionsHandler.GetOptions)
```

## 验收测试

### 测试用例

```bash
TOKEN=$(curl -s -X POST http://20.2.140.226:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  grep -o '"token":"[^"]*"' | cut -d'"' -f4)

curl -s -X GET "http://20.2.140.226:8080/api/v1/domains/options" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 测试结果

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 9039,
        "domain": "wx40.xyz",
        "status": "active",
        "purpose": "cdn",
        "createdAt": "2026-01-28T15:35:45.905+08:00",
        "updatedAt": "2026-01-28T16:03:14.452+08:00"
      },
      {
        "id": 9018,
        "domain": "4pxtech.com",
        "status": "active",
        "purpose": "cdn",
        "createdAt": "2026-01-24T10:50:40.507+08:00",
        "updatedAt": "2026-01-28T08:22:25.681+08:00"
      }
    ]
  },
  "message": "success"
}
```

### 验收结果

1. code=0：通过
2. data.items 存在：通过
3. 只返回 status=active 且 purpose=cdn 的域名：通过
4. 所有字段使用 lowerCamelCase：通过
5. 按 id 降序排列：通过（9039 > 9018）

## 接口规范

### 请求

```
GET /api/v1/domains/options
Authorization: Bearer <JWT_TOKEN>
```

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 9039,
        "domain": "wx40.xyz",
        "status": "active",
        "purpose": "cdn",
        "createdAt": "2026-01-28T15:35:45.905+08:00",
        "updatedAt": "2026-01-28T16:03:14.452+08:00"
      }
    ]
  }
}
```

### 字段说明

- id: 域名 ID
- domain: 域名
- status: 状态（active）
- purpose: 用途（cdn）
- createdAt: 创建时间
- updatedAt: 更新时间

## 交付物

1. DTO 文件：
   - api/v1/domains/dto.go

2. Handler 文件：
   - api/v1/domains/handler_options.go

3. 路由配置：
   - api/v1/router.go（已修改）

4. 交付报告：
   - DELIVERY_REPORT_C1-02.md

5. 代码已推送到 GitHub 仓库：labubu-daydayone/go_cmdb

6. 服务已部署并运行在测试服务器：http://20.2.140.226:8080

## 前端集成示例

### React + Ant Design

```typescript
import { Select } from 'antd';
import { useEffect, useState } from 'react';

interface DomainOption {
  id: number;
  domain: string;
  status: string;
  purpose: string;
  createdAt: string;
  updatedAt: string;
}

const DomainSelector = () => {
  const [domains, setDomains] = useState<DomainOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchDomains();
  }, []);

  const fetchDomains = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/domains/options', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      const result = await response.json();
      if (result.code === 0) {
        setDomains(result.data.items);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Select
      loading={loading}
      placeholder="请选择域名"
      options={domains.map(d => ({
        label: d.domain,
        value: d.id,
      }))}
    />
  );
};
```

## 注意事项

1. 接口只返回 CDN 用途的活跃域名
2. 按 id 降序排列，最新的域名在前
3. 所有字段命名遵循 lowerCamelCase 规范
4. 返回结构使用 data.items，符合项目规范

## 回滚策略

如需回滚：

1. 代码回滚：
   ```bash
   git revert <commit_hash>
   ```

2. 删除新增文件：
   - api/v1/domains/dto.go
   - api/v1/domains/handler_options.go

3. 恢复路由文件：
   - api/v1/router.go

4. 重新编译部署
