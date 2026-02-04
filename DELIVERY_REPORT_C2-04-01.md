# 交付报告：C2-04-01 - 控制端回源 Upstream 配置渲染与下发

**任务编号**：C2-04-01
**任务名称**：控制端回源 Upstream 配置渲染与下发
**任务级别**：P0

## 一、核心实现

本次任务完成了控制端回源配置从渲染到下发的完整闭环，为后续 Agent 端配置生效奠定了基础。主要实现内容包括：

1.  **Upstream 渲染服务**：
    -   创建了 `internal/upstream/renderer.go` 服务，负责将 `origin_set` 的配置渲染成 Nginx 的 `upstream` 配置块。
    -   实现了从 `origin_set_items` 表的 `snapshot_json` 字段中解析回源地址列表，并根据 `role` 和 `weight` 生成对应的 `server` 指令。

2.  **节点选择服务**：
    -   创建了 `internal/upstream/node_selector.go` 服务，用于根据 Website 的配置选择目标发布节点。
    -   支持按 `line_group_id` 选择节点，如果未指定，则默认选择所有在线节点。

3.  **发布服务与任务下发**：
    -   创建了 `internal/upstream/publisher.go` 服务，作为发布流程的入口。
    -   在发布时，会依次创建 `apply_config` 和 `reload` 两种类型的 `agent_tasks`。
    -   为 `reload` 任务增加了 10 秒的去抖控制，避免短时间内重复下发 `reload` 指令。
    -   同时创建 `release_tasks` 和 `release_task_nodes` 记录，用于前端查询发布状态和进度。

4.  **接口实现**：
    -   在 `websites` 模块下新增了 `POST /api/v1/websites/bind-origin-set` 接口，用于将 Website 与 Origin Set 绑定，并触发一次完整的发布流程。
    -   改造了 `releases` 模块的查询接口，将 `GET /api/v1/releases/:id` 修改为 `GET /api/v1/releases/detail?id=xxx`，并统一返回结构为 `data.item`。

## 二、修改文件清单

-   `internal/upstream/renderer.go` (新增)
-   `internal/upstream/node_selector.go` (新增)
-   `internal/upstream/publisher.go` (新增)
-   `api/v1/websites/handler.go` (修改)
-   `api/v1/releases/handler.go` (修改)
-   `internal/release/query.go` (修改)
-   `api/v1/router.go` (修改)

## 三、验收说明

由于本任务为控制端实现，其完整功能依赖于 Agent 端接口（C2-04-02）的实现。在 Agent 端接口完成前，无法进行完整的端到端验收。

当前已完成的验收包括：

-   **编译通过**：所有代码已在测试服务器上成功编译。
-   **接口可访问**：`POST /api/v1/websites/bind-origin-set` 和 `GET /api/v1/releases/detail` 接口已可访问，并能返回正确的错误信息（如测试数据不存在）。

**后续验收步骤（待 C2-04-02 完成后）**：

1.  **创建测试数据**：创建 Website、Origin Group、Origin Set 等测试数据。
2.  **调用绑定接口**：
    ```bash
    curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/bind-origin-set" \
    -H "Authorization: Bearer <JWT_TOKEN>" \
    -H "Content-Type: application/json" \
    -d '{"websiteId": <WEBSITE_ID>, "originSetId": <ORIGIN_SET_ID>}'
    ```
    预期返回 `releaseId`。

3.  **查询发布状态**：
    ```bash
    curl -s "http://20.2.140.226:8080/api/v1/releases/detail?id=<RELEASE_ID>" \
    -H "Authorization: Bearer <JWT_TOKEN>"
    ```
    预期返回完整的发布任务详情。

4.  **验证 Agent 任务**：在 `agent_tasks` 表中检查是否生成了对应的 `apply_config` 和 `reload` 任务。

5.  **验证 Agent 端配置**：在 Agent 端通过 `runtime/upstreams` 接口检查配置是否已下发并正确渲染。

## 四、回滚方案

如需回滚，可执行以下操作：

```bash
git revert <commit_hash>
```

其中 `<commit_hash>` 为本次任务提交的 Git Commit 哈希值。回滚后，重新编译并部署服务即可恢复到修改前状态。
