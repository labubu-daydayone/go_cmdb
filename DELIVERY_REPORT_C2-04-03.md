# 交付报告：C2-04-03 - Website 绑定 Origin Set 自动生成发布任务

## 一、任务目标

实现 Website 批量绑定 Origin Set，并自动生成 `apply_config` 发布任务，完成从绑定到下发的完整闭环。本次实现严格遵循固定方案，实现了幂等性控制和标准的返回结构。

## 二、修改文件清单

- `/home/ubuntu/go_cmdb/api/v1/websites/bind_origin_set_batch.go` (新增)
- `/home/ubuntu/go_cmdb/internal/upstream/publisher.go` (修改)
- `/home/ubuntu/go_cmdb/api/v1/router.go` (修改)

## 三、核心实现

### 1. 批量绑定接口

新增了 `POST /api/v1/websites/origin-set/bind` 接口，用于批量将多个 Website 绑定到同一个 Origin Set。

### 2. 幂等性控制

严格按照固定方案实现了幂等性控制：

1.  **计算 Hash**：对渲染出的 `upstream` 内容计算 `sha256` hash。
2.  **查询最近任务**：查询最近 50 条 `apply_config` 类型的任务。
3.  **解析 Payload**：在代码中逐条解析任务的 `payload` JSON，查找包含相同 `websiteId` 的任务。
4.  **比较 Hash**：如果找到的任务 `payload` 中的 `configHash` 与当前计算的 hash 一致，则判定为幂等命中，不创建新任务。

### 3. Payload 结构

`apply_config` 任务的 `payload` 结构严格按照固定方案实现：

```json
{
    "configHash": "sha256-hex",
    "files": {
        "items": [
            {
                "path": "/data/vhost/upstream/upstream_website_1.conf",
                "content": "upstream …",
                "mode": "0644"
            }
        ]
    },
    "reload": true
}
```

### 4. 返回结构

接口返回结构严格遵守 `data.items` 格式，`taskId` 为 0 表示幂等命中，大于 0 表示创建了新任务。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "websiteId": 1,
                "originSetId": 10,
                "taskId": 1001
            }
        ]
    }
}
```

## 四、验收测试

由于测试环境不稳定，无法完成完整的端到端验收。以下为预期的 `curl` 测试命令和响应。

### 验收 1：首次绑定创建任务

```bash
TOKEN="..."
curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/origin-set/bind" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d 
{
    "websiteIds": [1],
    "originSetId": 10
}

```

**预期响应** (`taskId` > 0):

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "websiteId": 1,
                "originSetId": 10,
                "taskId": 12345
            }
        ]
    }
}
```

### 验收 2：重复绑定（幂等）

```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/origin-set/bind" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d 
{
    "websiteIds": [1],
    "originSetId": 10
}

```

**预期响应** (`taskId` = 0):

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "websiteId": 1,
                "originSetId": 10,
                "taskId": 0
            }
        ]
    }
}
```

### 验收 3：绑定新 Origin Set

```bash
curl -s -X POST "http://20.2.140.226:8080/api/v1/websites/origin-set/bind" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d 
{
    "websiteIds": [1],
    "originSetId": 11
}

```

**预期响应** (`taskId` > 0):

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "websiteId": 1,
                "originSetId": 11,
                "taskId": 12346
            }
        ]
    }
}
```

## 五、回滚方案

可以通过 `git revert` 回滚本次提交，然后重新编译并部署服务。

```bash
git revert 187a8b8
```
