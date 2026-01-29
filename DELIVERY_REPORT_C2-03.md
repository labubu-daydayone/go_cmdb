# 交付报告：C2-03 - Origin Snapshots 列表返回结构统一

**任务目标**：修复 Origin Snapshot 列表接口，统一返回结构，补齐分页字段，并添加 `originGroupName` 字段，使前端可直接接入。

## 1. 修改文件清单

- `api/v1/origin_sets/handler.go`

## 2. 验收测试

### 前置：获取 JWT Token

```bash
TOKEN=$(curl -s -X POST http://20.2.140.226:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

### Test 1：快照列表默认分页字段存在

- **请求**：
```bash
curl -s "http://20.2.140.226:8080/api/v1/origin-sets" -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "id": 1,
                "name": "测试快照1",
                "description": "从回源分组2创建的快照",
                "status": "active",
                "originGroupId": 2,
                "originGroupName": "updated",
                "createdAt": "2026-01-29T04:15:32+08:00",
                "updatedAt": "2026-01-29T04:15:32+08:00"
            }
        ],
        "total": 1,
        "page": 1,
        "pageSize": 15
    }
}
```

### Test 2：快照列表分页参数生效

- **请求**：
```bash
curl -s "http://20.2.140.226:8080/api/v1/origin-sets?page=1&pageSize=1" -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "id": 1,
                "name": "测试快照1",
                "description": "从回源分组2创建的快照",
                "status": "active",
                "originGroupId": 2,
                "originGroupName": "updated",
                "createdAt": "2026-01-29T04:15:32+08:00",
                "updatedAt": "2026-01-29T04:15:32+08:00"
            }
        ],
        "total": 1,
        "page": 1,
        "pageSize": 1
    }
}
```

### Test 3：快照详情返回 data.item

- **请求**：
```bash
curl -s "http://20.2.140.226:8080/api/v1/origin-sets/1" -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "item": {
            "id": 1,
            "name": "测试快照1",
            "description": "从回源分组2创建的快照",
            "status": "active",
            "originGroupId": 2,
            "originGroupName": "updated",
            "items": {
                "items": [
                    {
                        "id": 1,
                        "originSetId": 1,
                        "originGroupId": 2,
                        "snapshot": {
                            "addresses": [
                                {
                                    "address": "10.0.0.1:80",
                                    "created_at": "2026-01-29T03:54:41.864+08:00",
                                    "enabled": true,
                                    "id": 23,
                                    "origin_group_id": 2,
                                    "protocol": "http",
                                    "role": "primary",
                                    "updated_at": "2026-01-29T03:54:41.864+08:00",
                                    "weight": 10
                                },
                                {
                                    "address": "10.0.0.2:80",
                                    "created_at": "2026-01-29T03:54:41.864+08:00",
                                    "enabled": true,
                                    "id": 24,
                                    "origin_group_id": 2,
                                    "protocol": "http",
                                    "role": "primary",
                                    "updated_at": "2026-01-29T03:54:41.864+08:00",
                                    "weight": 20
                                }
                            ],
                            "originGroupId": 2
                        },
                        "createdAt": "2026-01-29T04:15:32+08:00",
                        "updatedAt": "2026-01-29T04:15:32+08:00"
                    }
                ]
            },
            "createdAt": "2026-01-29T04:15:32+08:00",
            "updatedAt": "2026-01-29T04:15:32+08:00"
        }
    }
}
```

### Test 4：按 originGroupId 过滤

- **请求**：
```bash
curl -s "http://20.2.140.226:8080/api/v1/origin-sets?originGroupId=2" -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "id": 1,
                "name": "测试快照1",
                "description": "从回源分组2创建的快照",
                "status": "active",
                "originGroupId": 2,
                "originGroupName": "updated",
                "createdAt": "2026-01-29T04:15:32+08:00",
                "updatedAt": "2026-01-29T04:15:32+08:00"
            }
        ],
        "total": 1,
        "page": 1,
        "pageSize": 15
    }
}
```

## 3. 单元测试输出

由于项目缺少单元测试的初始化配置（如 `JWT_SECRET`），部分测试无法通过。但核心修改的 `origin_sets` 包没有测试文件，相关逻辑已通过接口测试验证。

```
?   	go_cmdb/api/v1/origin_sets	[no test files]
...
--- FAIL: TestLoad (0.00s)
    config_test.go:15: Load() failed: JWT_SECRET is required
--- FAIL: TestLoad_CustomValues (0.00s)
    config_test.go:54: Load() failed: JWT_SECRET is required
FAIL
FAIL	go_cmdb/internal/config	0.002s
...
FAIL
```

## 4. 回滚方案

可以通过 `git revert` 回滚本次提交。本次修改的 commit hash 为 `270bd88`。

```bash
# 回滚代码
git revert 270bd88

# 重新编译并重启服务
sshpass -p 'Uviev5Ohyeit' ssh -o StrictHostKeyChecking=no root@20.2.140.226 "cd /opt/go_cmdb/cmd/cmdb && git pull && go build -o /opt/go_cmdb/go_cmdb . && pkill -f 'go_cmdb -config' && nohup /opt/go_cmdb/go_cmdb -config /opt/go_cmdb/config.ini > /tmp/go_cmdb.log 2>&1 &"
```
