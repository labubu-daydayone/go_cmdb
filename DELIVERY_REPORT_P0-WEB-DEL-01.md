# 交付报告：P0-WEB-DEL-01 修复 Websites 删除失败

**任务目标**：修复因错误级联删除 Origin Set 导致的网站删除失败问题，确保网站删除操作的独立性和幂等性。

## 1. 实现概述

本次任务成功修复了 `POST /api/v1/websites/delete` 接口的错误逻辑。核心改动是**移除了删除网站时对 Origin Set 的级联删除操作**，使其符合“引用关系而非所有权关系”的设计原则。

**修复前**：删除网站会尝试删除其引用的 Origin Set，如果 Origin Set 因被其他资源引用而无法删除，则整个网站删除事务失败，返回 `5002: failed to delete origin set` 错误。

**修复后**：删除网站的逻辑调整为只清理网站自身及其直接附属资源（如 `website_domains`, `website_https`），不再触碰 Origin Set。这保证了网站删除操作的原子性和独立性，即使其引用的 Origin Set 仍然存在，网站也能被成功删除。

### 1.1. 修改文件清单

-   `api/v1/websites/handler.go`

## 2. 验收测试记录

所有验收标准均已通过 `curl` 命令验证，功能符合预期。

### 2.1. 验收 1：删除存在的网站必须成功

-   **请求**：
    ```bash
    curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/delete" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{"ids":[3]}'
    ```
-   **响应**：
    ```json
    {
        "code": 0,
        "message": "success",
        "data": null
    }
    ```

### 2.2. 验收 2：重复删除同一网站必须幂等成功

-   **请求**（再次执行）：
    ```bash
    curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/delete" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{"ids":[3]}'
    ```
-   **响应**：
    ```json
    {
        "code": 0,
        "message": "success",
        "data": null
    }
    ```

### 2.3. 验收 3：Origin Set 未被删除

删除网站（ID=3）后，其引用的 Origin Set（ID=1）仍然存在于数据库中，验证了级联删除已被移除。

-   **验证命令**：
    ```bash
    sshpass -p 'Uviev5Ohyeit' ssh root@20.2.140.226 "mysql -uadmin -pdeag2daicahThiipheed4gi4 -h20.2.140.226 cdn_control -e 'SELECT id, name FROM origin_sets WHERE id = 1;'"
    ```
-   **验证结果**：
    ```
    +----+-------------+
    | id | name        |
    +----+-------------+
    |  1 | 测试快照1   |
    +----+-------------+
    ```

## 3. 结论

任务 P0-WEB-DEL-01 已成功完成。网站删除接口的错误级联逻辑已修复，接口行为符合幂等性要求，并通过了所有验收标准。代码已提交至 GitHub 仓库。

## 4. 回滚方案

如需回滚，可 revert 本次任务的所有相关 commits。

```bash
# Revert the main feature commit
git revert 6c4fda9

# Re-deploy the service
# ... (compile and restart service)
```
