# API 文档：回源分组 (Origin Groups)

## 1. 查询回源分组列表

- **方法**: `GET`
- **路径**: `/api/v1/origin-groups/list`
- **Query 参数**:
  - `page` (number, optional, default: 1)
  - `pageSize` (number, optional, default: 10)

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 2,
        "name": "og-test-c2-01",
        "description": "C2-01 test origin group",
        "status": "active",
        "primaryCount": 2,
        "backupCount": 1,
        "enabledPrimaryCount": 2,
        "createdAt": "2026-01-29T02:19:38+08:00",
        "updatedAt": "2026-01-29T02:19:38+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

## 2. 查询回源分组详情

- **方法**: `GET`
- **路径**: `/api/v1/origin-groups/detail`
- **Query 参数**:
  - `id` (number, required)

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 2,
      "name": "og-test-c2-01",
      "description": "C2-01 test origin group",
      "status": "active",
      "createdAt": "2026-01-29T02:19:38+08:00",
      "updatedAt": "2026-01-29T02:19:38+08:00",
      "addresses": {
        "items": [
          {
            "id": 16,
            "address": "10.0.0.1:80",
            "role": "primary",
            "weight": 10,
            "enabled": true,
            "createdAt": "2026-01-29T02:45:28+08:00",
            "updatedAt": "2026-01-29T02:45:28+08:00"
          }
        ]
      }
    }
  }
}
```

## 3. 创建回源分组

- **方法**: `POST`
- **路径**: `/api/v1/origin-groups/create`
- **Body**:

```json
{
  "name": "my-new-origin-group",
  "description": "This is a new group"
}
```

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 3,
      "name": "my-new-origin-group",
      "description": "This is a new group",
      "status": "active",
      "createdAt": "2026-01-29T03:00:00+08:00",
      "updatedAt": "2026-01-29T03:00:00+08:00"
    }
  }
}
```

## 4. 批量更新地址

- **方法**: `POST`
- **路径**: `/api/v1/origin-groups/addresses/upsert`
- **Body**:

```json
{
  "originGroupId": 2,
  "items": [
    {
      "address": "1.1.1.1:80",
      "role": "primary",
      "weight": 10,
      "enabled": true
    },
    {
      "address": "2.2.2.2:80",
      "role": "backup",
      "weight": 999,
      "enabled": true
    }
  ]
}
```

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 2,
      "name": "og-test-c2-01",
      "description": "C2-01 test origin group",
      "status": "active",
      "createdAt": "2026-01-29T02:19:38+08:00",
      "updatedAt": "2026-01-29T03:05:00+08:00"
    }
  }
}
```

## 5. 更新回源分组

- **方法**: `POST`
- **路径**: `/api/v1/origin-groups/update`
- **Body**:

```json
{
  "id": 2,
  "name": "updated-group-name",
  "description": "Updated description"
}
```

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 2,
      "name": "updated-group-name",
      "description": "Updated description",
      "status": "active",
      "createdAt": "2026-01-29T02:19:38+08:00",
      "updatedAt": "2026-01-29T03:10:00+08:00"
    }
  }
}
```

## 6. 删除回源分组

- **方法**: `POST`
- **路径**: `/api/v1/origin-groups/delete`
- **Body**:

```json
{
  "id": 2
}
```

- **成功响应 (200)**:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```
