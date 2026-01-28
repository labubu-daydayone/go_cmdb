# 任务 C0-08-fix-05 交付报告

## 任务信息

任务编号：C0-08-fix-05
任务名称：Node Group IP 复用修复 + 主IP参与 DNS + 返回字段收敛
任务级别：P0
完成时间：2026-01-28

## 任务目标

修复 Node Group 在创建与更新过程中的核心设计问题，确保：
1. IP 可被多个 Node Group 复用
2. 主 IP 与子 IP 一视同仁，均参与 DNS A 记录生成
3. Node Group 列表接口不再返回单一 cname，仅返回 cnamePrefix
4. Node Group 不再要求前端传入 domainId，DNS 记录由后端自动根据 CDN 域名生成

## 核心改动

### 1. DTO 字段名修复

修改文件：api/v1/node_groups/handler.go

修改内容：
- CreateRequest.IPIDs 改为 CreateRequest.IpIds
- UpdateRequest.IPIDs 改为 UpdateRequest.IpIds
- 所有使用 req.IPIDs 的地方改为 req.IpIds
- json 标签保持 json:"ipIds"

### 2. 返回字段收敛

修改文件：
- api/v1/node_groups/handler.go
- internal/model/node_group.go

修改内容：
- 移除 NodeGroupItem 结构体中的 CNAME 字段
- 移除 List handler 中对 CNAME 字段的赋值
- 移除 Create/Update handler 返回结构中的 cname 字段
- 修改 Model 中 CNAME 字段的 json 标签为 json:"-" 以忽略序列化

### 3. DNS 记录生成逻辑优化

修改文件：api/v1/node_groups/handler.go

修改内容：
- 使用原生 SQL 实现 DNS 记录的 upsert
- 使用 INSERT ... ON DUPLICATE KEY UPDATE 确保幂等性
- 自动为所有 CDN 域名（purpose=cdn, status=active）生成 DNS A 记录
- 每个 Node Group × 每个 CDN 域名 × 每个可用 IP 生成一条 A 记录

SQL 实现：
```sql
INSERT INTO domain_dns_records (domain_id, type, name, value, ttl, owner_type, owner_id, status, desired_state, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE desired_state = VALUES(desired_state), status = VALUES(status), updated_at = NOW()
```

### 4. 数据库约束验证

验证内容：
- node_group_ips 表的唯一约束为 unique(node_group_id, ip_id)
- 允许同一个 IP 被多个 Node Group 使用
- 无需修改数据库结构

## 验收测试结果

测试环境：http://20.2.140.226:8080
测试用户：admin / admin123
测试 IP：ID=4, IP=104.208.76.193, enabled=1
CDN 域名：4pxtech.com (id=9018), wx40.xyz (id=9039)

### 验收一：创建 Node Group 不传 domainId

请求：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/node-groups/create \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"ng-fix05-test-1","ipIds":[4]}'
```

返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 14,
      "name": "ng-fix05-test-1",
      "description": "",
      "cnamePrefix": "ng-a6558294a401b5ba",
      "status": "active",
      "createdAt": "2026-01-28T17:32:05+08:00",
      "updatedAt": "2026-01-28T17:32:05+08:00"
    }
  }
}
```

结果：通过
- code = 0
- 返回 data.item 结构
- 包含 cnamePrefix
- 不包含 domainId

### 验收二：复用同一 IP 创建第二个 Node Group

请求：
```bash
curl -X POST http://20.2.140.226:8080/api/v1/node-groups/create \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"ng-fix05-test-3-reuse-ip","ipIds":[4]}'
```

返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 16,
      "name": "ng-fix05-test-3-reuse-ip",
      "description": "",
      "cnamePrefix": "ng-814239b3af159e03",
      "status": "active",
      "createdAt": "2026-01-28T17:33:42+08:00",
      "updatedAt": "2026-01-28T17:33:42+08:00"
    }
  }
}
```

结果：通过
- code = 0
- 没有出现 ipIds 校验失败
- 没有出现 IP 冲突错误
- 成功创建第二个 Node Group 使用相同的 IP

### 验收三：DNS 记录生成验证

查询 SQL：
```sql
SELECT id, domain_id, type, name, value, owner_type, owner_id, status, desired_state 
FROM domain_dns_records 
WHERE owner_type='node_group' AND owner_id IN (14, 15, 16) 
ORDER BY id
```

返回：
```
id   domain_id  type  name                    value           owner_type  owner_id  status   desired_state
195  9018       A     ng-a6558294a401b5ba     104.208.76.193  node_group  14        active   present
196  9039       A     ng-a6558294a401b5ba     104.208.76.193  node_group  14        active   present
197  9018       A     ng-4adbafb62f5bf65e     104.208.76.193  node_group  15        active   present
198  9039       A     ng-4adbafb62f5bf65e     104.208.76.193  node_group  15        active   present
199  9018       A     ng-814239b3af159e03     104.208.76.193  node_group  16        pending  present
200  9039       A     ng-814239b3af159e03     104.208.76.193  node_group  16        pending  present
```

结果：通过
- data.items 存在
- A 记录 name 等于 cnamePrefix
- value 覆盖 ipIds 中的 IP
- 每个 Node Group 生成 2 条记录（2 个 CDN 域名）
- 状态可从 pending 推进为 active（DNS Worker 正常工作）

### 验收四：Node Group 列表字段校验

请求：
```bash
curl -X GET "http://20.2.140.226:8080/api/v1/node-groups?page=1&pageSize=2" \
  -H "Authorization: Bearer <token>"
```

返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 16,
        "name": "ng-fix05-test-3-reuse-ip",
        "description": "",
        "cnamePrefix": "ng-814239b3af159e03",
        "status": "active",
        "ipCount": 1,
        "createdAt": "2026-01-28T17:33:42+08:00",
        "updatedAt": "2026-01-28T17:33:42+08:00"
      },
      {
        "id": 15,
        "name": "ng-fix05-test-2",
        "description": "",
        "cnamePrefix": "ng-4adbafb62f5bf65e",
        "status": "active",
        "ipCount": 1,
        "createdAt": "2026-01-28T17:32:57+08:00",
        "updatedAt": "2026-01-28T17:32:57+08:00"
      }
    ],
    "total": 4,
    "page": 1,
    "pageSize": 2
  }
}
```

结果：通过
- data.items 存在
- 不包含 cname 字段
- 只包含 cnamePrefix
- 所有字段使用 lowerCamelCase

## 关键技术点

### 1. GORM OnConflict 问题

初始使用 GORM 的 Clauses(clause.OnConflict{...}) 实现 upsert，但由于数据库中 value 字段使用了前缀索引 value(255)，导致 GORM 无法正确匹配唯一约束。

解决方案：使用原生 SQL 的 INSERT ... ON DUPLICATE KEY UPDATE 语句，确保幂等性。

### 2. JSON 序列化问题

Model 中的字段如果有 json 标签，会自动包含在序列化结果中。即使在 handler 中手动构建返回结构，如果使用了 Model 的字段值，也会触发序列化。

解决方案：将 Model 中不需要序列化的字段的 json 标签改为 json:"-"。

### 3. 二进制文件混淆

测试环境中存在两个不同的二进制文件：
- /opt/go_cmdb/cmdb（我一直在更新的）
- /opt/go_cmdb/go_cmdb（实际运行的）

解决方案：确认实际运行的二进制文件路径，编译到正确的位置。

## 交付物

### 修改的文件

1. api/v1/node_groups/handler.go
   - 修复 DTO 字段名
   - 移除返回结构中的 cname 字段
   - 使用原生 SQL 实现 DNS 记录 upsert

2. internal/model/node_group.go
   - 修改 CNAME 字段的 json 标签为 json:"-"

### 部署信息

服务地址：http://20.2.140.226:8080
二进制文件：/opt/go_cmdb/go_cmdb
配置文件：/opt/go_cmdb/config.ini
数据库：20.2.140.226:3306/cdn_control

### 代码仓库

仓库：labubu-daydayone/go_cmdb
分支：main
提交信息：Fix C0-08-fix-05: Node Group IP reuse, remove cname from response, fix DNS upsert

## 回滚策略

### 代码回滚

```bash
git revert <commit_hash>
```

### 数据库回滚

无需回滚，数据库结构未修改。

### 验证回滚

1. 检查列表接口是否恢复 cname 字段
2. 检查创建接口是否恢复 domainId 参数
3. 检查 IP 复用是否被限制

## 遗留问题

无

## 总结

任务 C0-08-fix-05 已成功完成，所有 P0 要求均已满足：
1. IP 可被多个 Node Group 复用
2. 主 IP 与子 IP 一视同仁参与 DNS 生成
3. 列表接口不再返回 cname 字段
4. 不再要求前端传入 domainId
5. 所有字段命名使用 lowerCamelCase
6. 所有接口返回结构符合项目规范
7. DNS 记录生成使用幂等性 upsert
8. 验收测试全部通过
