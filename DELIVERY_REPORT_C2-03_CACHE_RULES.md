# 交付报告：C2-03 缓存规则建模与管理接口

**任务目标**：实现一套独立的缓存规则管理能力，包括数据建模、规则组 CRUD、规则项管理等功能，为后续 Website 绑定缓存策略奠定基础。

## 1. 实现概述

本次任务成功实现了缓存规则（Cache Rules）模块的完整后端功能。主要工作包括：

1.  **数据建模**：创建了 `cache_rules`（规则组）和 `cache_rule_items`（规则项）两张核心数据表，并定义了相应的 GORM 模型。
2.  **接口实现**：开发了符合项目规范的 7 个 API 接口，覆盖了缓存规则组和规则项的创建、读取、更新、删除（CRUD）全部操作。
3.  **逻辑约束**：实现了 `name` 的唯一性约束、`match_type` + `match_value` 的组合唯一性约束，并为删除操作预留了对 Website 引用的检查点。
4.  **代码规范**：所有接口均使用 `GET`/`POST` 方法，返回结构严格遵守 `data.item` 和 `data.items` 规范，字段命名统一为 `lowerCamelCase`。

### 1.1. 修改文件清单

-   `internal/model/cache_rule.go` (新增)
-   `internal/model/cache_rule_item.go` (新增)
-   `api/v1/cache_rules/handler.go` (新增)
-   `api/v1/cache_rules/items_handler.go` (新增)
-   `api/v1/router.go` (修改)
-   `migrations/cache_rules.sql` (新增)

## 2. 接口实现详情

| HTTP 方法 | 路由                                      | 功能                 |
| :-------- | :---------------------------------------- | :------------------- |
| `POST`    | `/api/v1/cache-rules/create`              | 创建缓存规则组       |
| `GET`     | `/api/v1/cache-rules`                     | 获取缓存规则组列表   |
| `POST`    | `/api/v1/cache-rules/update`              | 更新缓存规则组       |
| `POST`    | `/api/v1/cache-rules/delete`              | 删除缓存规则组       |
| `POST`    | `/api/v1/cache-rules/items/upsert`        | 批量新增/更新规则项  |
| `GET`     | `/api/v1/cache-rules/{id}/items`          | 获取规则组下的规则项 |
| `POST`    | `/api/v1/cache-rules/items/delete`        | 删除规则项           |

## 3. 验收测试记录

所有接口均已通过 `curl` 命令验证，功能符合预期。

### 3.1. 创建缓存规则组

-   **请求**：
    ```bash
    curl -s -X POST "http://20.2.140.226:8080/api/v1/cache-rules/create" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{"name": "图片缓存规则", "enabled": true}'
    ```
-   **响应**：
    ```json
    {
        "code": 0,
        "message": "success",
        "data": {
            "item": {
                "id": 1,
                "name": "图片缓存规则",
                "enabled": true,
                "createdAt": "2026-02-04T20:26:39+08:00",
                "updatedAt": "2026-02-04T20:26:39+08:00"
            }
        }
    }
    ```

### 3.2. 缓存规则组列表

-   **请求**：
    ```bash
    curl -s "http://20.2.140.226:8080/api/v1/cache-rules" \
    -H "Authorization: Bearer <JWT_TOKEN>"
    ```
-   **响应**：
    ```json
    {
        "code": 0,
        "message": "success",
        "data": {
            "items": [
                {
                    "id": 1,
                    "name": "图片缓存规则",
                    "enabled": true,
                    "itemCount": 0,
                    "createdAt": "2026-02-04T20:26:39+08:00",
                    "updatedAt": "2026-02-04T20:26:39+08:00"
                }
            ],
            "total": 1,
            "page": 1,
            "pageSize": 15
        }
    }
    ```

### 3.3. 新增/更新规则项

-   **请求**：
    ```bash
    curl -s -X POST "http://20.2.140.226:8080/api/v1/cache-rules/items/upsert" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{
        "cacheRuleId": 1,
        "items": [
            {"matchType": "path", "matchValue": "/images/", "ttlSeconds": 3600, "enabled": true},
            {"matchType": "suffix", "matchValue": ".jpg", "ttlSeconds": 86400, "enabled": true}
        ]
    }'
    ```
-   **响应**：
    ```json
    {"code":0,"message":"success","data":null}
    ```

### 3.4. 获取规则项列表

-   **请求**：
    ```bash
    curl -s "http://20.2.140.226:8080/api/v1/cache-rules/1/items" \
    -H "Authorization: Bearer <JWT_TOKEN>"
    ```
-   **响应**：
    ```json
    {
        "code": 0,
        "message": "success",
        "data": {
            "items": [
                {
                    "id": 1,
                    "matchType": "path",
                    "matchValue": "/images/",
                    "ttlSeconds": 3600,
                    "enabled": true,
                    "createdAt": "2026-02-04T20:27:06+08:00",
                    "updatedAt": "2026-02-04T20:27:06+08:00"
                },
                {
                    "id": 2,
                    "matchType": "suffix",
                    "matchValue": ".jpg",
                    "ttlSeconds": 86400,
                    "enabled": true,
                    "createdAt": "2026-02-04T20:27:06+08:00",
                    "updatedAt": "2026-02-04T20:27:06+08:00"
                }
            ]
        }
    }
    ```

### 3.5. 删除规则组

-   **请求**：
    ```bash
    # First, create a rule to delete
    # ... (omitted for brevity)

    curl -s -X POST "http://20.2.140.226:8080/api/v1/cache-rules/delete" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{"ids":[2]}'
    ```
-   **响应**：
    ```json
    {"code":0,"message":"success","data":null}
    ```

## 4. 结论

任务 C2-03 已成功完成。缓存规则模块的后端基础功能已全部实现并通过验收，为后续的 Website 集成做好了准备。所有代码和数据库变更均已提交至 GitHub 仓库。

## 5. 回滚方案

如需回滚，可 revert 本次任务的所有相关 commits。

```bash
# Revert the main feature commit
git revert ab9a4b2

# Drop the database tables
sshpass -p 'Uviev5Ohyeit' ssh root@20.2.140.226 "mysql -uadmin -pdeag2daicahThiipheed4gi4 -h20.2.140.226 cdn_control -e 'DROP TABLE IF EXISTS cache_rule_items; DROP TABLE IF EXISTS cache_rules;'"

# Re-deploy the service
# ... (compile and restart service)
```
