# Line Group API 文档

本文档详细说明了 Line Group（线路组）模块的所有 API 接口，包括其功能、请求方式、参数及响应格式。

## 1. 获取线路组列表

该接口用于分页、筛选查询线路组列表。

-   **Method**: `GET`
-   **Path**: `/api/v1/line-groups`

### 请求参数 (Query)

| 字段          | 类型   | 是否必须 | 描述                               |
| :------------ | :----- | :------- | :--------------------------------- |
| `page`        | number | 否       | 页码，默认为 1                     |
| `pageSize`    | number | 否       | 每页数量，默认为 15                |
| `name`        | string | 否       | 线路组名称，支持模糊查询           |
| `domainId`    | number | 否       | 按对外域名 ID 筛选                 |
| `nodeGroupId` | number | 否       | 按节点组 ID 筛选                   |
| `status`      | string | 否       | 按状态筛选（例如 `active`, `inactive`） |

### 响应 (200 OK)

成功时返回线路组列表及分页信息。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "id": 4,
                "name": "lg-c1-04-test",
                "domainId": 9018,
                "domainName": "4pxtech.com",
                "nodeGroupId": 21,
                "nodeGroupName": "ng-c1-04-test",
                "cnamePrefix": "lg-3a179a8029cab5ec",
                "cname": "lg-3a179a8029cab5ec.4pxtech.com",
                "status": "active",
                "createdAt": "2026-01-28T21:24:14+08:00",
                "updatedAt": "2026-01-28T21:24:14+08:00"
            }
        ],
        "total": 1,
        "page": 1,
        "pageSize": 15
    }
}
```

## 2. 创建线路组

该接口用于创建一个新的线路组，并自动为其生成 CNAME 前缀和 DNS 解析记录。

-   **Method**: `POST`
-   **Path**: `/api/v1/line-groups/create`

### 请求体 (JSON)

| 字段          | 类型   | 是否必须 | 描述             |
| :------------ | :----- | :------- | :--------------- |
| `name`        | string | 是       | 线路组名称，需唯一 |
| `domainId`    | number | 是       | 关联的对外域名 ID  |
| `nodeGroupId` | number | 是       | 关联的节点组 ID    |

### 响应 (200 OK)

成功时返回新创建的线路组的完整信息。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "item": {
            "id": 4,
            "name": "lg-c1-04-test",
            "domainId": 9018,
            "domainName": "4pxtech.com",
            "nodeGroupId": 21,
            "nodeGroupName": "ng-c1-04-test",
            "cnamePrefix": "lg-3a179a8029cab5ec",
            "cname": "lg-3a179a8029cab5ec.4pxtech.com",
            "status": "active",
            "createdAt": "2026-01-28T21:24:14+08:00",
            "updatedAt": "2026-01-28T21:24:14+08:00"
        }
    }
}
```

## 3. 更新线路组

该接口用于更新一个已存在的线路组。可以更新名称、状态或更换关联的节点组。

-   **Method**: `POST`
-   **Path**: `/api/v1/line-groups/update`

### 请求体 (JSON)

| 字段          | 类型   | 是否必须 | 描述                                         |
| :------------ | :----- | :------- | :------------------------------------------- |
| `id`          | number | 是       | 要更新的线路组 ID                            |
| `name`        | string | 否       | 新的线路组名称，需唯一                       |
| `status`      | string | 否       | 新的状态（例如 `active`, `inactive`）        |
| `nodeGroupId` | number | 否       | 新的节点组 ID。更换会触发 DNS 记录的更新。 |

### 响应 (200 OK)

成功时返回更新后的线路组的完整信息。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "item": {
            "id": 4,
            "name": "lg-c1-04-test-updated",
            "domainId": 9018,
            "domainName": "4pxtech.com",
            "nodeGroupId": 21,
            "nodeGroupName": "ng-c1-04-test",
            "cnamePrefix": "lg-3a179a8029cab5ec",
            "cname": "lg-3a179a8029cab5ec.4pxtech.com",
            "status": "active",
            "createdAt": "2026-01-28T21:24:14+08:00",
            "updatedAt": "2026-01-28T21:24:42+08:00"
        }
    }
}
```

## 4. 删除线路组

该接口用于批量删除一个或多个线路组。

-   **Method**: `POST`
-   **Path**: `/api/v1/line-groups/delete`

### 请求体 (JSON)

| 字段 | 类型          | 是否必须 | 描述              |
| :--- | :------------ | :------- | :---------------- |
| `ids`  | array[number] | 是       | 要删除的线路组 ID 列表 |

### 响应 (200 OK)

成功时返回 `data` 为 `null`。

```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 5. 修复线路组 CNAME

该接口用于修复指定线路组的 CNAME 解析记录，确保其指向正确的、包含完整域名的节点组 CNAME。

-   **Method**: `POST`
-   **Path**: `/api/v1/line-groups/dns/repair-cname`

### 请求体 (JSON)

| 字段          | 类型   | 是否必须 | 描述              |
| :------------ | :----- | :------- | :---------------- |
| `lineGroupId` | number | 是       | 需要修复的线路组 ID |

### 响应 (200 OK)

成功时返回修复操作的详细信息。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "item": {
            "lineGroupId": 3,
            "domainId": 9018,
            "domain": "4pxtech.com",
            "expectedValue": "ng-684a169815130452.4pxtech.com",
            "affected": 1
        }
    }
}
```
