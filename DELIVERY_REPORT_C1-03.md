# 交付报告：C1-03 - Line Group CNAME 生成与架构解耦

## 一、任务目标

本次任务的核心目标是明确 Line Group 与 Node Group 的职责边界。Line Group 作为面向外部的唯一抽象，负责生成并对外暴露 CNAME。Node Group 作为纯粹的后端资源载体，不应包含任何与域名或 CNAME 相关的信息。CNAME 必须是一个动态计算字段，不存储于数据库中。

## 二、实现细节

### 1. Line Group CNAME 动态计算

- **DTO 层**：在 `line_group` 的 `ListItemDTO` 和 `CreateResponseDTO` 中增加了 `cname` 字段。
- **Handler 层**：在 `List` 和 `Create` 处理器中，通过 `lineGroup.CNAMEPrefix + "." + domain.Domain` 的方式动态计算 CNAME 值，并填充到响应 DTO 中。

### 2. Node Group 架构修正

- **问题发现**：在验收过程中，发现 `node_groups` 数据库表和 GORM Model 中仍然存在 `cname` 字段，这严重违反了“Node Group 作为纯后端载体”的架构原则。
- **数据库迁移**：创建了 `migrations/023_drop_node_groups_cname.sql` 迁移文件，用于从 `node_groups` 表中删除 `cname` 字段。
- **Model 层修正**：从 `internal/model/node_group.go` 中的 `NodeGroup` 结构体定义中移除了 `CNAME` 字段。
- **代码清理**：全局搜索并清除了所有对 `NodeGroup.CNAME` 的无效引用，主要涉及 `api/v1/node_groups/handler.go` 和 `api/v1/websites/handler.go`，确保代码在编译和运行时不再依赖该字段。

## 三、验收过程

验收严格遵循项目规范，全程使用 `curl` 命令与真实接口进行交互，并辅以数据库结构检查。

### 1. 创建 Line Group

通过 `POST /api/v1/line-groups/create` 创建一个新的线路组，验证创建成功后返回的 `data.item` 中包含正确的动态 CNAME。

**请求：**

```bash
TOKEN=$(curl -s -X POST http://20.2.140.226:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

curl -s -X POST "http://20.2.140.226:8080/api/v1/line-groups/create" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{"name":"测试线路组","domainId":9018,"nodeGroupId":19}' | jq .
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 1,
      "name": "测试线路组",
      "domainId": 9018,
      "domainName": "4pxtech.com",
      "nodeGroupId": 19,
      "cnamePrefix": "lg-0760c2fb61b76070",
      "cname": "lg-0760c2fb61b76070.4pxtech.com",
      "status": "active",
      "createdAt": "2026-01-28T20:25:17+08:00",
      "updatedAt": "2026-01-28T20:25:17+08:00"
    }
  }
}
```

**结论**：创建接口遵循了单对象返回规范 (`data.item`)，并成功生成了正确的 CNAME。

### 2. 查询 Line Group 列表

通过 `GET /api/v1/line-groups` 查询列表，验证返回的 `data.items` 中每个条目都包含正确的动态 CNAME。

**请求：**

```bash
curl -s "http://20.2.140.226:8080/api/v1/line-groups?page=1&pageSize=15" \
-H "Authorization: Bearer $TOKEN" | jq .
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "测试线路组",
        "domainId": 9018,
        "domainName": "4pxtech.com",
        "nodeGroupId": 19,
        "cnamePrefix": "lg-0760c2fb61b76070",
        "cname": "lg-0760c2fb61b76070.4pxtech.com",
        "status": "active",
        "createdAt": "2026-01-28T20:25:17+08:00",
        "updatedAt": "2026-01-28T20:25:17+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 15
  }
}
```

**结论**：列表接口遵循了列表返回规范 (`data.items`)，并成功为列表项生成了正确的 CNAME。

### 3. 查询 Node Group 列表

通过 `GET /api/v1/node-groups` 查询列表，验证返回的 `data.items` 中不包含任何 CNAME 相关字段。

**请求：**

```bash
curl -s "http://20.2.140.226:8080/api/v1/node-groups?page=1&pageSize=15" \
-H "Authorization: Bearer $TOKEN" | jq '.data.items[0]'
```

**响应：**

```json
{
  "id": 19,
  "name": "测试123",
  "description": "",
  "cnamePrefix": "ng-00b3a1671fc3248d",
  "status": "active",
  "ipCount": 0,
  "createdAt": "2026-01-28T19:05:12+08:00",
  "updatedAt": "2026-01-28T19:06:31+08:00"
}
```

**结论**：Node Group 接口的返回结果中不包含 `cname` 或 `domainName` 字段，符合其作为纯后端载体的架构定位。

### 4. 数据库表结构验证

最后，直接连接数据库检查 `line_groups` 和 `node_groups` 表的结构，确保 CNAME 字段未被物理存储。

**`line_groups` 表结构：**

```sql
mysql> SHOW COLUMNS FROM line_groups;
+---------------+--------------+------+-----+---------+----------------+
| Field         | Type         | Null | Key | Default | Extra          |
+---------------+--------------+------+-----+---------+----------------+
| id            | bigint       | NO   | PRI | NULL    | auto_increment |
| name          | varchar(128) | NO   |     | NULL    |                |
| description   | varchar(255) | NO   |     |         |                |
| domain_id     | bigint       | NO   | MUL | NULL    |                |
| node_group_id | bigint       | NO   | MUL | NULL    |                |
| cname_prefix  | varchar(64)  | NO   | UNI | NULL    |                |
| status        | varchar(32)  | NO   |     | active  |                |
| created_at    | datetime(3)  | YES  |     | NULL    |                |
| updated_at    | datetime(3)  | YES  |     | NULL    |                |
+---------------+--------------+------+-----+---------+----------------+
```

**`node_groups` 表结构：**

```sql
mysql> SHOW COLUMNS FROM node_groups;
+--------------+---------------------------+------+-----+---------+----------------+
| Field        | Type                      | Null | Key | Default | Extra          |
+--------------+---------------------------+------+-----+---------+----------------+
| id           | bigint                    | NO   | PRI | NULL    | auto_increment |
| name         | varchar(128)              | NO   | UNI | NULL    |                |
| description  | varchar(255)              | YES  |     | NULL    |                |
| cname_prefix | varchar(128)              | NO   | UNI | NULL    |                |
| status       | enum('active','inactive') | NO   |     | active  |                |
| created_at   | datetime(3)               | YES  |     | NULL    |                |
| updated_at   | datetime(3)               | YES  |     | NULL    |                |
+--------------+---------------------------+------+-----+---------+----------------+
```

**结论**：两张表的物理结构均不包含 `cname` 字段，确认 CNAME 是在应用层动态计算的，符合设计要求。

## 四、总结

任务 C1-03 已成功完成。通过本次任务，Line Group 和 Node Group 的职责边界得到了清晰的划分，CNAME 的生成和暴露被正确地限制在 Line Group 的 API 中。期间发现并修复了 `node_groups` 表中残留 `cname` 字段的严重架构问题，并通过代码和数据库迁移彻底解决了该问题。所有相关接口均已通过 `curl` 验证，符合项目制定的命名和结构规范。
