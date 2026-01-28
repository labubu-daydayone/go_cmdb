# 交付报告：C2-02 - Origin Set 快照创建与绑定接口

**任务目标**：实现 Origin Set 快照的创建、列表、详情和绑定 Website 的功能。

## 1. 实现概览

### 1.1. 数据模型

- **`origin_sets` 表**：修改了现有表，增加了 `name`, `description`, `status` 字段。
- **`origin_set_items` 表**：新建了该表，用于存储快照的具体内容（冻结的 JSON）。

### 1.2. API 接口

- **`POST /api/v1/origin-sets/create`**：创建 Origin Set 快照。
- **`GET /api/v1/origin-sets`**：获取 Origin Set 列表。
- **`GET /api/v1/origin-sets/:id`**：获取 Origin Set 详情。
- **`POST /api/v1/origin-sets/bind-website`**：将 Origin Set 绑定到 Website。

## 2. 验收过程

### 2.1. 创建 Origin Set 快照

- **请求**：
```bash
curl -s -X POST http://20.2.140.226:8080/api/v1/origin-sets/create -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"测试快照1","description":"从回源分组2创建的快照","originGroupId":2}'
```

- **响应**：
```json
{"code":0,"message":"success","data":{"item":{"id":1,"name":"测试快照1","description":"从回源分组2创建的快照","status":"active","originGroupId":2,"createdAt":"2026-01-29T04:15:32+08:00","updatedAt":"2026-01-29T04:15:32+08:00"}}}
```

### 2.2. 查询 Origin Set 列表

- **请求**：
```bash
curl -s -X GET http://20.2.140.226:8080/api/v1/origin-sets -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{"code":0,"message":"success","data":{"items":[{"id":1,"name":"测试快照1","description":"从回源分组2创建的快照","status":"active","originGroupId":2,"createdAt":"2026-01-29T04:15:32+08:00","updatedAt":"2026-01-29T04:15:32+08:00"}]}}
```

### 2.3. 查询 Origin Set 详情

- **请求**：
```bash
curl -s -X GET http://20.2.140.226:8080/api/v1/origin-sets/1 -H "Authorization: Bearer $TOKEN"
```

- **响应**：
```json
{"code":0,"message":"success","data":{"item":{"id":1,"name":"测试快照1","description":"从回源分组2创建的快照","status":"active","originGroupId":2,"items":{"items":[{"id":1,"originSetId":1,"originGroupId":2,"snapshot":{"addresses":[{"address":"10.0.0.1:80","created_at":"2026-01-29T03:54:41.864+08:00","enabled":true,"id":23,"origin_group_id":2,"protocol":"http","role":"primary","updated_at":"2026-01-29T03:54:41.864+08:00","weight":10},{"address":"10.0.0.2:80","created_at":"2026-01-29T03:54:41.864+08:00","enabled":true,"id":24,"origin_group_id":2,"protocol":"http","role":"primary","updated_at":"2026-01-29T03:54:41.864+08:00","weight":20}],"originGroupId":2},"createdAt":"2026-01-29T04:15:32+08:00","updatedAt":"2026-01-29T04:15:32+08:00"}]},"createdAt":"2026-01-29T04:15:32+08:00","updatedAt":"2026-01-29T04:15:32+08:00"}}}
```

### 2.4. 绑定 Website

- **请求**：
```bash
curl -s -X POST http://20.2.140.226:8080/api/v1/origin-sets/bind-website -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"websiteId":3,"originSetId":1}'
```

- **响应**：
```json
{"code":0,"message":"success","data":null}
```

- **数据库验证**：
```sql
SELECT id, origin_set_id FROM websites WHERE id=3;
```
```
+----+---------------+
| id | origin_set_id |
+----+---------------+
|  3 |             1 |
+----+---------------+
```

## 3. 结论

所有接口均已实现并通过验收，符合任务要求。
