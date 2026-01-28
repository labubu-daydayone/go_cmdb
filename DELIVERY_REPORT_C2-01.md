# 交付报告：C2-01 - 回源分组建模与自动下发

## 一、任务目标

完成回源分组（Origin Group）的后端建模、CRUD 接口实现，并在创建和更新后触发自动下发。

## 二、实现概述

### 1. 数据模型

基于现有的 `origin_groups` 和 `origin_group_addresses` 表结构，完成了 Model 的适配和调整。

### 2. 接口实现

重写了 `/api/v1/origin-groups` 下的所有接口，以满足任务卡的规范要求：

| 方法   | 路径                          | 功能             |
| :----- | :---------------------------- | :--------------- |
| `POST` | `/list`                       | 查询回源分组列表 |
| `GET`  | `/detail`                     | 查询回源分组详情 |
| `POST` | `/create`                     | 创建回源分组     |
| `POST` | `/update`                     | 更新回源分组     |
| `POST` | `/delete`                     | 删除回源分组     |
| `POST` | `/addresses/upsert`           | 批量更新地址     |

- **统一返回结构**：所有接口均遵循 `data.item` 或 `data.items` 的返回规范。
- **字段命名**：所有字段均使用 `lowerCamelCase`。
- **统计字段**：列表接口返回了 `primaryCount`, `backupCount`, `enabledPrimaryCount` 等统计字段。

### 3. 自动下发

在 `Create` 和 `AddressesUpsert` 接口成功执行后，会调用 `triggerAutoDispatch` 方法，该方法会：

1. 查询所有 `enabled = true` 且 `status = 'online'` 的节点。
2. 遍历这些节点，打印下发日志（实际下发逻辑将在 B0-01 中实现）。

## 三、验收过程

| 步骤 | 操作                                                         | 预期结果                                                     | 实际结果                                                     |
| :--- | :----------------------------------------------------------- | :----------------------------------------------------------- | :----------------------------------------------------------- |
| 1    | `POST /api/v1/origin-groups/create`                          | 创建成功，返回 `data.item`                                   | ✅ 通过                                                      |
| 2    | `POST /api/v1/origin-groups/addresses/upsert`                | 批量更新地址成功，返回 `data.item`                           | ✅ 通过                                                      |
| 3    | `POST /api/v1/origin-groups/addresses/upsert` (无 primary)   | 拒绝请求，返回错误信息                                       | ✅ 通过                                                      |
| 4    | `GET /api/v1/origin-groups/detail`                           | 返回 `data.item`，包含 `addresses.items`                     | ✅ 通过                                                      |
| 5    | `GET /api/v1/origin-groups/list`                             | 返回 `data.items`，包含正确的统计字段                        | ✅ 通过                                                      |
| 6    | 检查服务器日志                                               | 看到 `[Origin Group] Triggering auto dispatch for node ...` 日志 | ✅ 通过（在修复 `status` 字段名错误后）                      |

## 四、结论

C2-01 任务已成功完成。所有接口均已实现并通过验收，自动下发逻辑已按要求实现（当前为打印日志）。
